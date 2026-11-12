package prompts

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRegisterNoPrompts(t *testing.T) {
	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	if err := Register(s); err != nil {
		t.Fatalf("Register with no prompts failed: %v", err)
	}
}

func TestPromptHandlerIsCalled(t *testing.T) {
	called := false
	prompt := Prompt{
		Name:        "debug_cluster",
		Description: "Systematic cluster investigation",
		Arguments: []Argument{
			{Name: "cluster_id", Description: "The cluster to investigate", Required: true},
		},
		Handler: func(ctx context.Context, args map[string]string) ([]*mcp.PromptMessage, error) {
			called = true
			if args["cluster_id"] != "minikube" {
				return nil, errors.New("unexpected args")
			}
			return []*mcp.PromptMessage{
				{Role: mcp.Role("user"), Content: &mcp.TextContent{Text: "Investigate cluster minikube"}},
			}, nil
		},
	}

	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	if err := register(s, prompt); err != nil {
		t.Fatalf("register failed: %v", err)
	}

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

	res, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name:      "debug_cluster",
		Arguments: map[string]string{"cluster_id": "minikube"},
	})
	if err != nil {
		t.Fatalf("GetPrompt failed: %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
	if res == nil || len(res.Messages) == 0 {
		t.Fatal("no result messages")
	}
	text, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || text.Text != "Investigate cluster minikube" {
		t.Errorf("unexpected result: %+v", res.Messages)
	}
}
