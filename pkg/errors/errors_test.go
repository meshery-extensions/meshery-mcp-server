package errors

import (
	"encoding/json"
	"testing"
)

func TestMeshKitErrors_Contract(t *testing.T) {
	errAuth := ErrUnauthenticated(nil)
	if errAuth == nil {
		t.Fatalf("expected ErrUnauthenticated error, got nil")
	}

	errUpstream := ErrUpstreamFailed(nil)
	if errUpstream == nil {
		t.Fatalf("expected ErrUpstreamFailed error, got nil")
	}

	errSchema := ErrSchemaInvalid(nil)
	if errSchema == nil {
		t.Fatalf("expected ErrSchemaInvalid error, got nil")
	}
}

func TestMeshKitErrors_JSONWireFormat(t *testing.T) {
	errAuth := ErrUnauthenticated(nil)
	rawJSON, err := json.Marshal(errAuth)
	if err != nil {
		t.Fatalf("failed to marshal MeshKit error to JSON: %v", err)
	}

	var wire struct {
		Code                 string   `json:"Code"`
		Severity             int      `json:"Severity"`
		ShortDescription     []string `json:"ShortDescription"`
		LongDescription      []string `json:"LongDescription"`
		ProbableCause        []string `json:"ProbableCause"`
		SuggestedRemediation []string `json:"SuggestedRemediation"`
	}

	if err := json.Unmarshal(rawJSON, &wire); err != nil {
		t.Fatalf("failed to unmarshal MeshKit error JSON into typed struct: %v", err)
	}

	if wire.Code != ErrUnauthenticatedCode {
		t.Errorf("expected Code %s, got %s", ErrUnauthenticatedCode, wire.Code)
	}

	if len(wire.ShortDescription) == 0 || wire.ShortDescription[0] != "Authentication Failure" {
		t.Errorf("expected ShortDescription array with Authentication Failure, got: %v", wire.ShortDescription)
	}

	if len(wire.SuggestedRemediation) == 0 {
		t.Errorf("expected SuggestedRemediation array to be non-empty, got: %v", wire.SuggestedRemediation)
	}
}
