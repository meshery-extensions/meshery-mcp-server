package tools

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"regexp"
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

// List methods report total, or len(items) when total is 0.
type fakeBackend struct {
	components    []componentv1beta3.ComponentDefinition
	component     componentv1beta3.ComponentDefinition
	models        []modelv1beta1.ModelDefinition
	model         modelv1beta1.ModelDefinition
	relationships []relationshipv1alpha3.RelationshipDefinition
	total         int
	err           error

	calls    int
	gotPage  Page
	gotQuery string
	gotModel string
	gotID    uuid.UUID
}

func result[T any](items []T, total int) Result[T] {
	if total == 0 {
		total = len(items)
	}
	return Result[T]{Items: items, Total: total}
}

func (f *fakeBackend) SearchComponents(_ context.Context, query, model string, p Page) (Result[componentv1beta3.ComponentDefinition], error) {
	f.calls++
	f.gotQuery, f.gotModel, f.gotPage = query, model, p
	return result(f.components, f.total), f.err
}

func (f *fakeBackend) GetComponent(_ context.Context, id uuid.UUID) (componentv1beta3.ComponentDefinition, error) {
	f.calls++
	f.gotID = id
	return f.component, f.err
}

func (f *fakeBackend) ListModels(_ context.Context, _ string, p Page) (Result[modelv1beta1.ModelDefinition], error) {
	f.calls++
	f.gotPage = p
	return result(f.models, f.total), f.err
}

func (f *fakeBackend) GetModel(_ context.Context, id uuid.UUID) (modelv1beta1.ModelDefinition, error) {
	f.calls++
	f.gotID = id
	return f.model, f.err
}

func (f *fakeBackend) ListRelationships(_ context.Context, model string, p Page) (Result[relationshipv1alpha3.RelationshipDefinition], error) {
	f.calls++
	f.gotModel, f.gotPage = model, p
	return result(f.relationships, f.total), f.err
}

func mustUUID(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.FromString(s)
	if err != nil {
		t.Fatalf("parsing UUID %q: %v", s, err)
	}
	return id
}

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

// The current revision and the last one with an initialize handshake.
var protocolVersions = []string{"2026-07-28", "2025-11-25"}

func connect(t *testing.T, b Backend, protocolVersion string) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	s := newServer()
	if _, err := Register(s, b); err != nil {
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
		backend  Backend
		wantCode string
	}{
		{name: "nil server", server: nil, backend: &fakeBackend{}, wantCode: ErrServerRequiredCode},
		{name: "nil backend", server: newServer(), backend: nil, wantCode: ErrBackendRequiredCode},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Register(tt.server, tt.backend)
			wantError(t, err, tt.wantCode)
		})
	}
}

func listTools(t *testing.T, protocolVersion string) []*mcp.Tool {
	t.Helper()
	res, err := connect(t, &fakeBackend{}, protocolVersion).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	return res.Tools
}

// A duplicate name would silently replace an earlier tool.
func TestRegister_ReturnsWhatItServes(t *testing.T) {
	names, err := Register(newServer(), &fakeBackend{})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("Register returned no tool names")
	}

	var served []string
	for _, tool := range listTools(t, protocolVersions[0]) {
		served = append(served, tool.Name)
	}
	slices.Sort(served)
	sorted := slices.Sorted(slices.Values(names))
	if !slices.Equal(sorted, served) {
		t.Errorf("Register returned %v, clients are served %v", sorted, served)
	}
	if compact := slices.Compact(slices.Clone(sorted)); len(compact) != len(sorted) {
		t.Errorf("Register returned duplicate names: %v", sorted)
	}
}

var toolName = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// Per-tool expectations are tested next to each tool, with wantTool.
func TestTools_Contract(t *testing.T) {
	for _, version := range protocolVersions {
		t.Run(version, func(t *testing.T) {
			for _, tool := range listTools(t, version) {
				if !toolName.MatchString(tool.Name) {
					t.Errorf("%q: name is not snake_case", tool.Name)
				}
				if tool.Title == "" || tool.Description == "" {
					t.Errorf("%s: want a title and a description", tool.Name)
				}
				if tool.Annotations == nil {
					t.Errorf("%s: want annotations, so clients know whether it changes anything", tool.Name)
				}
				for kind, schema := range map[string]any{"input": tool.InputSchema, "output": tool.OutputSchema} {
					if schemaType(t, schema) != "object" {
						t.Errorf("%s: %s schema type is not object", tool.Name, kind)
					}
				}
			}
		})
	}
}

type toolSpec struct {
	required       []string // required input arguments, in any order
	readOnly       bool
	notImplemented bool // its Meshery call does not exist yet
}

func wantTool(t *testing.T, name string, spec toolSpec) {
	t.Helper()
	for _, version := range protocolVersions {
		var tool *mcp.Tool
		for _, listed := range listTools(t, version) {
			if listed.Name == name {
				tool = listed
			}
		}
		if tool == nil {
			t.Fatalf("%s: tool %q is not served", version, name)
		}

		required := schemaRequired(t, tool.InputSchema)
		want := slices.Sorted(slices.Values(spec.required))
		if !slices.Equal(required, want) {
			t.Errorf("%s: required = %v, want %v", version, required, want)
		}
		if got := tool.Annotations != nil && tool.Annotations.ReadOnlyHint; got != spec.readOnly {
			t.Errorf("%s: ReadOnlyHint = %v, want %v", version, got, spec.readOnly)
		}
		if got := strings.Contains(tool.Description, notImplementedNote); got != spec.notImplemented {
			t.Errorf("%s: description says not implemented = %v, want %v", version, got, spec.notImplemented)
		}
	}
}

func schemaOf(t *testing.T, schema any) (typ string, required []string) {
	t.Helper()
	b, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	var s struct {
		Type     string   `json:"type"`
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	slices.Sort(s.Required)
	return s.Type, s.Required
}

func schemaType(t *testing.T, schema any) string {
	t.Helper()
	typ, _ := schemaOf(t, schema)
	return typ
}

func schemaRequired(t *testing.T, schema any) []string {
	t.Helper()
	_, required := schemaOf(t, schema)
	return required
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
		{name: "malformed id is rejected by the handler", tool: "get_model",
			args: map[string]any{"id": "m1"}, wantIsError: true, wantCalls: 0},
		{name: "search needs query or model", tool: "search_components",
			args: map[string]any{}, wantIsError: true, wantCalls: 0},
		{name: "page arguments are accepted", tool: "list_relationships",
			args: map[string]any{"page": 1, "pageSize": 10}, wantCalls: 1},
	}
	for _, version := range protocolVersions {
		for _, tt := range tests {
			t.Run(version+"/"+tt.name, func(t *testing.T) {
				fake := &fakeBackend{}
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

func TestParseID(t *testing.T) {
	const canonical = "5f4e2a1c-8d3b-4c6e-9a7f-1b2c3d4e5f60"
	tests := []struct {
		name     string
		value    string
		want     string // canonical form; "" = expect an error
		wantCode string
	}{
		{name: "canonical", value: canonical, want: canonical},
		{name: "upper case", value: strings.ToUpper(canonical), want: canonical},
		{name: "surrounding space", value: " " + canonical + "\n", want: canonical},
		{name: "braces", value: "{" + canonical + "}", want: canonical},
		{name: "urn prefix", value: "urn:uuid:" + canonical, want: canonical},
		{name: "no hyphens", value: strings.ReplaceAll(canonical, "-", ""), want: canonical},
		{name: "empty", value: "", wantCode: ErrRequiredArgumentCode},
		{name: "blank", value: "   ", wantCode: ErrRequiredArgumentCode},
		{name: "not a UUID", value: "c1", wantCode: ErrInvalidIDCode},
		{name: "truncated", value: canonical[:35], wantCode: ErrInvalidIDCode},
		{name: "all zero", value: "00000000-0000-0000-0000-000000000000", wantCode: ErrInvalidIDCode},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := parseID("id", tt.value)
			if tt.wantCode != "" {
				wantError(t, err, tt.wantCode, `"id"`)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id.String() != tt.want {
				t.Errorf("parseID(%q) = %s, want %s", tt.value, id, tt.want)
			}
		})
	}
}

func TestFromBackend(t *testing.T) {
	wrap := func(err error) error { return ErrListModels(err) }

	t.Run("MeshKit error passes through unchanged", func(t *testing.T) {
		original := errors.New("test-code", errors.Alert,
			[]string{"Short"}, []string{"Long"}, []string{"Cause"}, []string{"Remedy"})
		if got := fromBackend(original, wrap); got != original {
			t.Errorf("fromBackend = %v, want the original error", got)
		}
	})

	t.Run("other errors are wrapped", func(t *testing.T) {
		got := fromBackend(stderrors.New("connection refused"), wrap)
		wantError(t, got, ErrListModelsCode, "connection refused")
	})
}

func TestTools_RegistryErrorIsToolError(t *testing.T) {
	errUnavailable := stderrors.New("unavailable")

	for _, version := range protocolVersions {
		t.Run(version, func(t *testing.T) {
			cs := connect(t, &fakeBackend{err: errUnavailable}, version)

			res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      "get_component",
				Arguments: map[string]any{"id": componentID},
			})
			if err != nil {
				t.Fatalf("CallTool returned a protocol error: %v", err)
			}
			if !res.IsError || len(res.Content) == 0 {
				t.Fatalf("want a tool error the model can read, got %+v", res)
			}
			text, _ := res.Content[0].(*mcp.TextContent)
			if text == nil || !strings.Contains(text.Text, `Getting component "`+componentID+`" failed`) || !strings.Contains(text.Text, "unavailable") {
				t.Errorf("error text = %v, want the operation and the cause", res.Content)
			}
		})
	}
}
