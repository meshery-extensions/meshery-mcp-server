package resources

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRegisterNoResources(t *testing.T) {
	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	if err := Register(s); err != nil {
		t.Fatalf("Register with no resources failed: %v", err)
	}
}

func TestResourceHandlerIsCalled(t *testing.T) {
	const uri = "meshery://clusters/minikube/topology"
	called := false
	resource := Resource{
		URI:         uri,
		Name:        "Cluster topology",
		Description: "The discovered state of a cluster",
		MIMEType:    "application/json",
		Handler: func(ctx context.Context, u string) (string, error) {
			called = true
			if u != uri {
				return "", errors.New("unexpected uri")
			}
			return `{"nodes": 1}`, nil
		},
	}

	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	if err := registerResource(s, resource); err != nil {
		t.Fatalf("registerResource failed: %v", err)
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

	res, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
	if res == nil || len(res.Contents) == 0 {
		t.Fatal("no result contents")
	}
	content := res.Contents[0]
	if content.Text != `{"nodes": 1}` {
		t.Errorf("unexpected text: %q", content.Text)
	}
}

func TestTemplateHandlerIsCalled(t *testing.T) {
	called := false
	template := ResourceTemplate{
		URITemplate: "meshery://clusters/{cluster_id}/topology",
		Name:        "Cluster topology template",
		Description: "The discovered state of a cluster",
		MIMEType:    "application/json",
		Handler: func(ctx context.Context, uri string) (string, error) {
			called = true
			return `{"cluster": "minikube"}`, nil
		},
	}

	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	if err := registerTemplate(s, template); err != nil {
		t.Fatalf("registerTemplate failed: %v", err)
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

	res, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "meshery://clusters/minikube/topology"})
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
	if res == nil || len(res.Contents) == 0 {
		t.Fatal("no result contents")
	}
	if res.Contents[0].Text != `{"cluster": "minikube"}` {
		t.Errorf("unexpected text: %q", res.Contents[0].Text)
	}
}
