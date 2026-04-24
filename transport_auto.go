package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type framingMode int

const (
	modeUnknown framingMode = iota
	modeNDJSON
	modeHeader
)

// AutoTransport supports both NDJSON and Content-Length framed JSON-RPC.
// It auto-detects the framing based on the first inbound message.
type AutoTransport struct{}

func (*AutoTransport) Connect(context.Context) (mcp.Connection, error) {
	return newAutoConn(rwc{rc: os.Stdin, wc: nopWriteCloser{os.Stdout}}), nil
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

type rwc struct {
	rc io.ReadCloser
	wc io.WriteCloser
}

func (r rwc) Read(p []byte) (int, error)  { return r.rc.Read(p) }
func (r rwc) Write(p []byte) (int, error) { return r.wc.Write(p) }
func (r rwc) Close() error {
	rcErr := r.rc.Close()
	wcErr := r.wc.Close()
	return errors.Join(rcErr, wcErr)
}

type msgOrErr struct {
	msg json.RawMessage
	err error
}

type autoConn struct {
	rwc      io.ReadWriteCloser
	reader   *bufio.Reader
	dec      *json.Decoder
	mode     framingMode
	modeOnce sync.Once

	writeMu sync.Mutex

	incoming chan msgOrErr
	closed   chan struct{}
	closeMu  sync.Once
	closeErr error
}

func newAutoConn(rwc io.ReadWriteCloser) *autoConn {
	c := &autoConn{
		rwc:      rwc,
		reader:   bufio.NewReader(rwc),
		incoming: make(chan msgOrErr),
		closed:   make(chan struct{}),
	}
	go c.readLoop()
	return c
}

func (c *autoConn) SessionID() string { return "" }

func (c *autoConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case me := <-c.incoming:
		if me.err != nil {
			c.Close()
			return nil, me.err
		}
		return jsonrpc.DecodeMessage(me.msg)
	}
}

func (c *autoConn) Write(ctx context.Context, msg jsonrpc.Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	data, err := jsonrpc.EncodeMessage(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	mode := c.getMode()
	switch mode {
	case modeHeader:
		if _, err := fmt.Fprintf(c.rwc, "Content-Length: %d\r\n\r\n", len(data)); err != nil {
			return err
		}
		_, err = c.rwc.Write(data)
		return err
	case modeNDJSON, modeUnknown:
		if _, err := c.rwc.Write(data); err != nil {
			return err
		}
		_, err = c.rwc.Write([]byte("\n"))
		return err
	default:
		return fmt.Errorf("unknown framing mode")
	}
}

func (c *autoConn) Close() error {
	c.closeMu.Do(func() {
		close(c.closed)
		c.closeErr = c.rwc.Close()
	})
	return c.closeErr
}

func (c *autoConn) readLoop() {
	for {
		raw, err := c.readMessage()
		select {
		case c.incoming <- msgOrErr{msg: raw, err: err}:
		case <-c.closed:
			return
		}
		if err != nil {
			return
		}
	}
}

func (c *autoConn) readMessage() (json.RawMessage, error) {
	mode := c.getMode()
	switch mode {
	case modeHeader:
		return c.readHeaderMessage()
	case modeNDJSON:
		return c.readNDJSONMessage()
	default:
		// default to NDJSON if detection is inconclusive
		return c.readNDJSONMessage()
	}
}

func (c *autoConn) getMode() framingMode {
	c.modeOnce.Do(func() {
		c.mode = detectMode(c.reader)
	})
	return c.mode
}

func detectMode(r *bufio.Reader) framingMode {
	const headerPrefix = "Content-Length"
	peekSizes := []int{16, 64, 256}
	for _, n := range peekSizes {
		b, err := r.Peek(n)
		if len(b) == 0 && err != nil {
			break
		}
		i := 0
		for i < len(b) {
			ch := b[i]
			if ch != ' ' && ch != '\t' && ch != '\r' && ch != '\n' {
				break
			}
			i++
		}
		if i >= len(b) {
			if err != nil {
				break
			}
			continue
		}
		rest := b[i:]
		if len(rest) >= len(headerPrefix) && bytes.HasPrefix(rest, []byte(headerPrefix)) {
			return modeHeader
		}
		// anything else: assume NDJSON
		return modeNDJSON
	}
	return modeNDJSON
}

func (c *autoConn) readNDJSONMessage() (json.RawMessage, error) {
	if c.dec == nil {
		c.dec = json.NewDecoder(c.reader)
	}
	var raw json.RawMessage
	if err := c.dec.Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *autoConn) readHeaderMessage() (json.RawMessage, error) {
	var contentLength int64
	firstRead := true
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && firstRead && line == "" {
				return nil, io.EOF
			}
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return nil, fmt.Errorf("failed reading header line: %w", err)
		}
		firstRead = false
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		colon := strings.IndexRune(line, ':')
		if colon < 0 {
			return nil, fmt.Errorf("invalid header line %q", line)
		}
		name := line[:colon]
		value := strings.TrimSpace(line[colon+1:])
		if strings.EqualFold(name, "Content-Length") {
			n, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed parsing Content-Length: %v", value)
			}
			if n <= 0 {
				return nil, fmt.Errorf("invalid Content-Length: %v", n)
			}
			contentLength = n
		}
	}
	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	data := make([]byte, contentLength)
	if _, err := io.ReadFull(c.reader, data); err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}
