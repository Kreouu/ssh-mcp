package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func isValidEnvKey(key string) bool {
	return envKeyPattern.MatchString(key)
}

func shellEscape(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func expandUserPath(path string) string {
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	return path
}

type limitedBuffer struct {
	buf       bytes.Buffer
	remaining int64
	truncated bool
}

func newLimitedBuffer(limit int64) *limitedBuffer {
	return &limitedBuffer{remaining: limit}
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	if l.remaining <= 0 {
		l.truncated = true
		return len(p), nil
	}
	if int64(len(p)) > l.remaining {
		_, _ = l.buf.Write(p[:l.remaining])
		l.truncated = true
		l.remaining = 0
		return len(p), nil
	}
	_, err := l.buf.Write(p)
	l.remaining -= int64(len(p))
	return len(p), err
}

func (l *limitedBuffer) String() string {
	return l.buf.String()
}

func (l *limitedBuffer) Truncated() bool {
	return l.truncated
}

func joinRemotePath(base, rel string) string {
	if base == "" {
		return rel
	}
	if rel == "" {
		return base
	}
	base = strings.TrimSuffix(base, "/")
	rel = strings.TrimPrefix(rel, "/")
	return fmt.Sprintf("%s/%s", base, rel)
}
