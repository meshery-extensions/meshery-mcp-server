package tools

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"slices"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/meshery/meshkit/errors"
	relationshipv1alpha3 "github.com/meshery/schemas/models/v1alpha3/relationship"
	modelv1beta1 "github.com/meshery/schemas/models/v1beta1/model"
	componentv1beta3 "github.com/meshery/schemas/models/v1beta3/component"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// fakeRegistry implements Registry. Each method records that it was called
// and what it received, and returns the configured values.
type fakeRegistry struct {
	components    []componentv1beta3.ComponentDefinition
	component     componentv1beta3.ComponentDefinition
	models        []modelv1beta1.ModelDefinition
	model         modelv1beta1.ModelDefinition
	relationships []relationshipv1alpha3.RelationshipDefinition
	err           error

	calls       int
	gotPage     int
	gotPageSize int
	gotQuery    string
	gotKind     string
	gotModel    string
	gotID       string
}

func (f *fakeRegistry) SearchComponents(_ context.Context, query, kind, model string) ([]componentv1beta3.ComponentDefinition, error) {
	f.calls++
	f.gotQuery, f.gotKind, f.gotModel = query, kind, model
	return f.components, f.err
}

func (f *fakeRegistry) GetComponent(_ context.Context, id string) (componentv1beta3.ComponentDefinition, error) {
	f.calls++
	f.gotID = id
	return f.component, f.err
}

func (f *fakeRegistry) ListModels(_ context.Context, _ string, page, pageSize int) ([]modelv1beta1.ModelDefinition, error) {
	f.calls++
	f.gotPage, f.gotPageSize = page, pageSize
	return f.models, f.err
}

func (f *fakeRegistry) GetModel(_ context.Context, id string) (modelv1beta1.ModelDefinition, error) {
	f.calls++
	f.gotID = id
	return f.model, f.err
}

func (f *fakeRegistry) ListRelationships(_ context.Context, model string) ([]relationshipv1alpha3.RelationshipDefinition, error) {
	f.calls++
	f.gotModel = model
	return f.relationships, f.err
}

// mustUUID parses a UUID literal for test fixtures.
func mustUUID(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.FromString(s)
	if err != nil {
		t.Fatalf("parsing UUID %q: %v", s, err)
	}
	return id
}

// wantError checks that err is the MeshKit error with code and that its text
// contains each of texts, for example the cause it wraps.
func wantError(t *testing.T, err error, code string, texts ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want error %s, got nil", code)
	}
	if got := errors.GetCode(err); got != code {
		t.Errorf("error code = %q, want %q (error: %v)", got, code, err)
	}
	for _, text := range texts {
		if !strings.Contains(err.Error(), text) {
			t.Errorf("error %q does not contain %q", err.Error(), text)
		}
	}
}

func newServer() *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.0.0"}, nil)
}

// protocolVersions are the two protocol eras clients use: the current
// sessionless protocol and the last one with an initialize handshake, which
// most hosts still speak.
var protocolVersions = []string{"2026-07-28", "2025-11-25"}

// connect registers the tools on a server and returns a client session
// connected to it over in-memory transports, speaking protocolVersion.
func connect(t *testing.T, r Registry, protocolVersion string) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	s := newServer()
	if _, err := Register(s, r); err != nil {
		t.Fatalf("Register: %v", err)
	}

	st, ct := mcp.NewInMemoryTransports()
	ss, err := s.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = ss.Close() })

	c := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, nil)
	cs, err := c.Connect(ctx, ct, &mcp.ClientSessionOptions{ProtocolVersion: protocolVersion})
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func TestRegister_RequiresDependencies(t *testing.T) {
	tests := []struct {
		name     string
		server   *mcp.Server
		registry Registry
		wantErr  string
	}{
		{name: "nil server", server: nil, registry: &fakeRegistry{}, wantErr: "MCP server is required to register tools"},
		{name: "nil registry", server: newServer(), registry: nil, wantErr: "Meshery registry is required to register tools"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Register(tt.server, tt.registry)
			if err == nil || errors.GetSDescription(err) != tt.wantErr {
				t.Fatalf("want error %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestRegister_ReturnsToolNames(t *testing.T) {
	names, err := Register(newServer(), &fakeRegistry{})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	want := []string{"search_components", "get_component", "list_models", "get_model", "list_relationships"}
	if !slices.Equal(names, want) {
		t.Errorf("names = %v, want %v", names, want)
	}
}

func TestTools_List(t *testing.T) {
	wantRequired := map[string][]string{
		"search_components":  {"query"},
		"get_component":      {"id"},
		"list_models":        nil,
		"get_model":          {"id"},
		"list_relationships": nil,
	}
	var want []string
	for name := range wantRequired {
		want = append(want, name)
	}
	slices.Sort(want)

	for _, version := range protocolVersions {
		t.Run(version, func(t *testing.T) {
			cs := connect(t, &fakeRegistry{}, version)

			res, err := cs.ListTools(context.Background(), nil)
			if err != nil {
				t.Fatalf("ListTools: %v", err)
			}

			var got []string
			for _, tool := range res.Tools {
				got = append(got, tool.Name)

				if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
					t.Errorf("%s: want ReadOnlyHint true", tool.Name)
				}
				if !strings.Contains(tool.Description, notImplementedNote) {
					t.Errorf("%s: description does not say the tool is not implemented", tool.Name)
				}

				required := requiredFields(t, tool.InputSchema)
				if !slices.Equal(required, wantRequired[tool.Name]) {
					t.Errorf("%s: required = %v, want %v", tool.Name, required, wantRequired[tool.Name])
				}
			}

			slices.Sort(got)
			if !slices.Equal(got, want) {
				t.Errorf("tools = %v, want %v", got, want)
			}
		})
	}
}

func requiredFields(t *testing.T, schema any) []string {
	t.Helper()
	b, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	var s struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	slices.Sort(s.Required)
	return s.Required
}

func TestTools_Call(t *testing.T) {
	tests := []struct {
		name        string
		tool        string
		args        map[string]any
		wantIsError bool
		wantCalls   int
	}{
		{name: "missing required argument is a tool error", tool: "get_component",
			args: map[string]any{}, wantIsError: true, wantCalls: 0},
		{name: "optional arguments may be omitted", tool: "search_components",
			args: map[string]any{"query": "deployment"}, wantCalls: 1},
		{name: "no arguments needed", tool: "list_models",
			args: map[string]any{}, wantCalls: 1},
		{name: "empty id is rejected by the handler", tool: "get_model",
			args: map[string]any{"id": ""}, wantIsError: true, wantCalls: 0},
	}
	for _, version := range protocolVersions {
		for _, tt := range tests {
			t.Run(version+"/"+tt.name, func(t *testing.T) {
				fake := &fakeRegistry{}
				cs := connect(t, fake, version)

				res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: tt.tool, Arguments: tt.args})
				if err != nil {
					t.Fatalf("CallTool returned a protocol error: %v", err)
				}
				if res.IsError != tt.wantIsError {
					t.Errorf("IsError = %v, want %v (content: %v)", res.IsError, tt.wantIsError, res.Content)
				}
				if fake.calls != tt.wantCalls {
					t.Errorf("registry calls = %d, want %d", fake.calls, tt.wantCalls)
				}
			})
		}
	}
}

func TestFromRegistry(t *testing.T) {
	wrap := func(err error) error { return ErrListModels(err) }

	t.Run("MeshKit error passes through unchanged", func(t *testing.T) {
		original := errors.New("test-code", errors.Alert,
			[]string{"Short"}, []string{"Long"}, []string{"Cause"}, []string{"Remedy"})
		if got := fromRegistry(original, wrap); got != original {
			t.Errorf("fromRegistry = %v, want the original error", got)
		}
	})

	t.Run("other errors are wrapped", func(t *testing.T) {
		got := fromRegistry(stderrors.New("connection refused"), wrap)
		wantError(t, got, ErrListModelsCode, "connection refused")
	})
}

func TestTools_RegistryErrorIsToolError(t *testing.T) {
	errUnavailable := stderrors.New("unavailable")

	for _, version := range protocolVersions {
		t.Run(version, func(t *testing.T) {
			cs := connect(t, &fakeRegistry{err: errUnavailable}, version)

			res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      "get_component",
				Arguments: map[string]any{"id": "c1"},
			})
			if err != nil {
				t.Fatalf("CallTool returned a protocol error: %v", err)
			}
			if !res.IsError || len(res.Content) == 0 {
				t.Fatalf("want a tool error the model can read, got %+v", res)
			}
			text, _ := res.Content[0].(*mcp.TextContent)
			if text == nil || !strings.Contains(text.Text, `Getting component "c1" failed`) || !strings.Contains(text.Text, "unavailable") {
				t.Errorf("error text = %v, want the operation and the cause", res.Content)
			}
		})
	}
}
