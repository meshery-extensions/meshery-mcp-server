package tools

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRegisterNoTools(t *testing.T) {
	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	if err := Register(s); err != nil {
		t.Fatalf("Register with no tools failed: %v", err)
	}
}

func TestToolHandlerIsCalled(t *testing.T) {
	called := false
	tool := Tool{
		Name:        "echo",
		Description: "echo the input",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{"type": "string"},
			},
		},
		Handler: func(ctx context.Context, args any) (any, error) {
			called = true
			m, ok := args.(map[string]any)
			if !ok {
				return nil, errors.New("args not a map")
			}
			return m["message"], nil
		},
	}

	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	if err := register(s, tool); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Drive the tool through an in-memory client session to confirm the
	// registration adapter wires the handler end to end.
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		_ = s.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client Connect failed: %v", err)
	}
	defer func() { _ = session.Close() }()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "echo", Arguments: map[string]any{"message": "hi"}})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
	if res == nil || len(res.Content) == 0 {
		t.Fatal("no result content")
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok || text.Text != "hi" {
		t.Errorf("unexpected result: %+v", res.Content)
	}
}
