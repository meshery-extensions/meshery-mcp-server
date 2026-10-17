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

// protocolVersions are the versions the server is tested with: 2026-07-28,
// the current stateless revision, and 2025-11-25, the last revision with an
// initialize handshake. The SDK serves both; serving the older one is
// optional under the 2026-07-28 versioning rules.
var protocolVersions = []string{"2026-07-28", "2025-11-25"}

// fakeRegistry implements tools.Registry and records the token each call saw.
type fakeRegistry struct {
	gotToken string
	gotOK    bool
}

func (f *fakeRegistry) record(ctx context.Context) {
	f.gotToken, f.gotOK = meshery.TokenFromContext(ctx)
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

func staticToken(token string) CredentialsFunc {
	return func(ctx context.Context, _ mcp.Request) (context.Context, error) {
		return meshery.WithToken(ctx, token), nil
	}
}

// start runs the server on one end of an in-memory transport and returns a
// client session speaking protocolVersion, the cancel func for the server's
// context, and a channel that receives the server's return value.
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
		Arguments: map[string]any{"id": "c1"},
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

	if _, err := newServer(Options{Name: "test", Registry: &fakeRegistry{}, Logger: logger}); err != nil {
		t.Fatalf("newServer: %v", err)
	}

	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var entry struct {
			Msg       string   `json:"msg"`
			Tools     []string `json:"tools"`
			Resources []string `json:"resources"`
			Prompts   []string `json:"prompts"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil || entry.Msg != "registered capabilities" {
			continue
		}
		want := []string{"search_components", "get_component", "list_models", "get_model", "list_relationships"}
		if !slices.Equal(entry.Tools, want) {
			t.Errorf("tools = %v, want %v", entry.Tools, want)
		}
		if entry.Resources == nil || entry.Prompts == nil {
			t.Errorf("want resources and prompts listed (empty), got %v and %v", entry.Resources, entry.Prompts)
		}
		return
	}
	t.Fatalf("no \"registered capabilities\" line in log output:\n%s", buf.String())
}

func TestCapabilities_OnlyTools(t *testing.T) {
	for _, version := range protocolVersions {
		t.Run(version, func(t *testing.T) {
			// A 2026-07-28 client reads capabilities from server/discover and
			// a 2025-11-25 client from initialize; the SDK client keeps
			// either result in InitializeResult.
			cs, _, _ := start(t, Options{Name: "test", Registry: &fakeRegistry{}}, version)

			res := cs.InitializeResult()
			if res == nil || res.Capabilities == nil {
				t.Fatal("no capabilities in the result")
			}
			// Compare what is sent on the wire: only the tools capability.
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
			cs, _, _ := start(t, Options{Name: "test", Registry: fake, Credentials: staticToken("abc")}, version)

			res, err := callGetComponent(cs)
			if err != nil || res.IsError {
				t.Fatalf("CallTool: err=%v result=%v", err, res)
			}
			if !fake.gotOK || fake.gotToken != "abc" {
				t.Errorf("registry saw token (%q, %v), want (%q, true)", fake.gotToken, fake.gotOK, "abc")
			}
		})

		t.Run(version+"/no credentials means no token", func(t *testing.T) {
			fake := &fakeRegistry{}
			cs, _, _ := start(t, Options{Name: "test", Registry: fake}, version)

			if _, err := callGetComponent(cs); err != nil {
				t.Fatalf("CallTool: %v", err)
			}
			if fake.gotOK {
				t.Errorf("registry saw token %q, want none", fake.gotToken)
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
