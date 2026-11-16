package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	relationshipv1alpha3 "github.com/meshery/schemas/models/v1alpha3/relationship"
	modelv1beta1 "github.com/meshery/schemas/models/v1beta1/model"
	componentv1beta3 "github.com/meshery/schemas/models/v1beta3/component"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
	"github.com/meshery-extensions/meshery-mcp-server/internal/tools"
)

var _ tools.Registry = (*fakeRegistry)(nil)

// The current revision and the last one with an initialize handshake.
var protocolVersions = []string{"2026-07-28", "2025-11-25"}

type fakeRegistry struct {
	gotCreds meshery.Credentials
	gotOK    bool
}

func (f *fakeRegistry) record(ctx context.Context) {
	f.gotCreds, f.gotOK = meshery.CredentialsFromContext(ctx)
}

func (f *fakeRegistry) SearchComponents(ctx context.Context, _, _, _ string) ([]componentv1beta3.ComponentDefinition, error) {
	f.record(ctx)
	return nil, nil
}

func (f *fakeRegistry) GetComponent(ctx context.Context, _ string) (componentv1beta3.ComponentDefinition, error) {
	f.record(ctx)
	return componentv1beta3.ComponentDefinition{}, nil
}

func (f *fakeRegistry) ListModels(ctx context.Context, _ string, _, _ int) ([]modelv1beta1.ModelDefinition, error) {
	f.record(ctx)
	return nil, nil
}

func (f *fakeRegistry) GetModel(ctx context.Context, _ string) (modelv1beta1.ModelDefinition, error) {
	f.record(ctx)
	return modelv1beta1.ModelDefinition{}, nil
}

func (f *fakeRegistry) ListRelationships(ctx context.Context, _ string) ([]relationshipv1alpha3.RelationshipDefinition, error) {
	f.record(ctx)
	return nil, nil
}

var testCreds = meshery.Credentials{Provider: "Meshery", Token: "abc"}

func staticCredentials(creds meshery.Credentials) CredentialsFunc {
	return func(ctx context.Context, _ mcp.Request) (context.Context, error) {
		return meshery.WithCredentials(ctx, creds), nil
	}
}

func start(t *testing.T, opts Options, protocolVersion string) (*mcp.ClientSession, context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	st, ct := mcp.NewInMemoryTransports()
	done := make(chan error, 1)
	go func() { done <- runTransport(ctx, opts, st) }()

	c := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, nil)
	cs, err := c.Connect(context.Background(), ct, &mcp.ClientSessionOptions{ProtocolVersion: protocolVersion})
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs, cancel, done
}

func wait(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop")
		return nil
	}
}

func callGetComponent(cs *mcp.ClientSession) (*mcp.CallToolResult, error) {
	return cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_component",
		Arguments: map[string]any{"id": "5f4e2a1c-8d3b-4c6e-9a7f-1b2c3d4e5f60"},
	})
}

func TestNewServer_RequiresRegistry(t *testing.T) {
	if _, err := newServer(Options{Name: "test"}); err == nil {
		t.Fatal("want error for missing registry, got nil")
	}
}

func TestNewServer_LogsCapabilities(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	server, err := newServer(Options{Name: "test", Registry: &fakeRegistry{}, Logger: logger})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	// Read the log before connecting: the session logs from another goroutine.
	logged := capabilitiesLine(t, buf.String())

	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	var served []string
	for _, tool := range res.Tools {
		served = append(served, tool.Name)
	}
	if !slices.Equal(slices.Sorted(slices.Values(logged.Tools)), slices.Sorted(slices.Values(served))) {
		t.Errorf("logged tools %v, served %v", logged.Tools, served)
	}
	if logged.Resources == nil || logged.Prompts == nil {
		t.Errorf("want resources and prompts listed (empty), got %v and %v", logged.Resources, logged.Prompts)
	}
}

type capabilitiesEntry struct {
	Msg       string   `json:"msg"`
	Tools     []string `json:"tools"`
	Resources []string `json:"resources"`
	Prompts   []string `json:"prompts"`
}

func capabilitiesLine(t *testing.T, log string) capabilitiesEntry {
	t.Helper()
	for _, line := range strings.Split(strings.TrimSpace(log), "\n") {
		var entry capabilitiesEntry
		if err := json.Unmarshal([]byte(line), &entry); err == nil && entry.Msg == "registered capabilities" {
			return entry
		}
	}
	t.Fatalf("no \"registered capabilities\" line in log output:\n%s", log)
	return capabilitiesEntry{}
}

func TestCapabilities_OnlyTools(t *testing.T) {
	for _, version := range protocolVersions {
		t.Run(version, func(t *testing.T) {
			// From server/discover (2026-07-28) or initialize (2025-11-25).
			cs, _, _ := start(t, Options{Name: "test", Registry: &fakeRegistry{}}, version)

			res := cs.InitializeResult()
			if res == nil || res.Capabilities == nil {
				t.Fatal("no capabilities in the result")
			}
			b, err := json.Marshal(res.Capabilities)
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]json.RawMessage
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatal(err)
			}
			keys := slices.Sorted(maps.Keys(got))
			if !slices.Equal(keys, []string{"tools"}) {
				t.Errorf("capabilities = %s, want only tools", b)
			}
		})
	}
}

func TestCredentials(t *testing.T) {
	for _, version := range protocolVersions {
		t.Run(version+"/token reaches the registry", func(t *testing.T) {
			fake := &fakeRegistry{}
			cs, _, _ := start(t, Options{Name: "test", Registry: fake, Credentials: staticCredentials(testCreds)}, version)

			res, err := callGetComponent(cs)
			if err != nil || res.IsError {
				t.Fatalf("CallTool: err=%v result=%v", err, res)
			}
			if !fake.gotOK || fake.gotCreds != testCreds {
				t.Errorf("backend saw credentials (%v, %v), want (%v, true)", fake.gotCreds, fake.gotOK, testCreds)
			}
		})

		t.Run(version+"/no credentials means no token", func(t *testing.T) {
			fake := &fakeRegistry{}
			cs, _, _ := start(t, Options{Name: "test", Registry: fake}, version)

			if _, err := callGetComponent(cs); err != nil {
				t.Fatalf("CallTool: %v", err)
			}
			if fake.gotOK {
				t.Errorf("backend saw credentials %v, want none", fake.gotCreds)
			}
		})

		t.Run(version+"/an error rejects the request", func(t *testing.T) {
			errDenied := errors.New("denied")
			deny := func(ctx context.Context, req mcp.Request) (context.Context, error) {
				if _, ok := req.(*mcp.CallToolRequest); ok {
					return nil, errDenied
				}
				return ctx, nil
			}
			cs, _, _ := start(t, Options{Name: "test", Registry: &fakeRegistry{}, Credentials: deny}, version)

			if _, err := callGetComponent(cs); err == nil {
				t.Fatal("want a protocol error, got nil")
			}
		})
	}
}

func TestRunTransport_CleanShutdown(t *testing.T) {
	for _, version := range protocolVersions {
		t.Run(version+"/context cancelled", func(t *testing.T) {
			_, cancel, done := start(t, Options{Name: "test", Registry: &fakeRegistry{}}, version)
			cancel()
			if err := wait(t, done); err != nil {
				t.Errorf("runTransport returned %v, want nil", err)
			}
		})

		t.Run(version+"/client disconnects", func(t *testing.T) {
			cs, _, done := start(t, Options{Name: "test", Registry: &fakeRegistry{}}, version)
			_ = cs.Close()
			if err := wait(t, done); err != nil {
				t.Errorf("runTransport returned %v, want nil", err)
			}
		})
	}
}
