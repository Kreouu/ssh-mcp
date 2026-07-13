package main

import (
	"context"
	"io"
	"log"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPServerExposesExpectedTools(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Name: "ssh-mcp", Version: "test"},
		Hosts: []HostConfig{{
			Name:    "dev-board",
			Address: "192.168.1.50",
			Port:    22,
			User:    "pi",
		}},
	}
	policy, err := NewCommandPolicy(PolicyConfig{}, false)
	if err != nil {
		t.Fatal(err)
	}
	app, err := NewApp(cfg, policy, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatal(err)
	}

	server := newMCPServer(cfg, app)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tools) != 4 {
		t.Fatalf("expected 4 tools, got %d", len(result.Tools))
	}

	expected := map[string]struct {
		readOnly    bool
		destructive bool
		openWorld   bool
	}{
		"ssh_list_hosts": {readOnly: true},
		"ssh_exec":       {destructive: true, openWorld: true},
		"ssh_upload":     {destructive: true, openWorld: true},
		"ssh_download":   {destructive: true, openWorld: true},
	}
	for _, tool := range result.Tools {
		want, ok := expected[tool.Name]
		if !ok {
			t.Fatalf("unexpected tool %q", tool.Name)
		}
		if tool.Description == "" || tool.InputSchema == nil || tool.OutputSchema == nil {
			t.Fatalf("tool %q is missing description or schema", tool.Name)
		}
		if tool.Annotations == nil {
			t.Fatalf("tool %q is missing annotations", tool.Name)
		}
		if tool.Annotations.ReadOnlyHint != want.readOnly {
			t.Fatalf("tool %q readOnly=%v, want %v", tool.Name, tool.Annotations.ReadOnlyHint, want.readOnly)
		}
		if tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint != want.destructive {
			t.Fatalf("tool %q has unexpected destructive annotation", tool.Name)
		}
		if tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint != want.openWorld {
			t.Fatalf("tool %q has unexpected open-world annotation", tool.Name)
		}
	}

	call, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ssh_list_hosts",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if call.IsError || len(call.Content) == 0 {
		t.Fatalf("unexpected ssh_list_hosts result: %#v", call)
	}
}
