package security

import (
	"encoding/json"
	"testing"
)

func TestSanitizeMap_SensitiveKeys(t *testing.T) {
	input := map[string]interface{}{
		"username": "admin",
		"password": "supersecretpassword123",
		"token":    "bearer-token-xyz-789",
		"cluster":  "k8s-prod-cluster",
	}

	sanitized := SanitizeMap(input)

	if sanitized["password"] != "[REDACTED_SECRET]" {
		t.Errorf("expected password to be redacted, got %v", sanitized["password"])
	}
	if sanitized["token"] != "[REDACTED_SECRET]" {
		t.Errorf("expected token to be redacted, got %v", sanitized["token"])
	}
	if sanitized["username"] != "admin" {
		t.Errorf("expected username to be admin, got %v", sanitized["username"])
	}
}

func TestSanitizeMap_NestedStructures(t *testing.T) {
	input := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "meshery-design",
			"credentials": map[string]interface{}{
				"kubeconfig": "apiVersion: v1...",
				"api_key":    "key-12345",
			},
		},
		"endpoints": []interface{}{
			map[string]interface{}{
				"url":   "https://mesh.local",
				"token": "secret-auth-token",
			},
		},
	}

	sanitized := SanitizeMap(input)

	metadata := sanitized["metadata"].(map[string]interface{})
	creds := metadata["credentials"].(map[string]interface{})

	if creds["kubeconfig"] != "[REDACTED_SECRET]" {
		t.Errorf("expected kubeconfig to be redacted, got %v", creds["kubeconfig"])
	}
	if creds["api_key"] != "[REDACTED_SECRET]" {
		t.Errorf("expected api_key to be redacted, got %v", creds["api_key"])
	}

	endpoints := sanitized["endpoints"].([]interface{})
	ep0 := endpoints[0].(map[string]interface{})
	if ep0["token"] != "[REDACTED_SECRET]" {
		t.Errorf("expected endpoint token to be redacted, got %v", ep0["token"])
	}
}

func TestSanitizeMap_CaseInsensitive(t *testing.T) {
	input := map[string]interface{}{
		"AuthToken":   "secret-123",
		"KubeConfig":  "cluster-config",
		"PASSWORD":    "pass-456",
		"publicField": "public-val",
	}

	sanitized := SanitizeMap(input)

	if sanitized["AuthToken"] != "[REDACTED_SECRET]" {
		t.Errorf("expected AuthToken to be redacted, got %v", sanitized["AuthToken"])
	}
	if sanitized["KubeConfig"] != "[REDACTED_SECRET]" {
		t.Errorf("expected KubeConfig to be redacted, got %v", sanitized["KubeConfig"])
	}
	if sanitized["PASSWORD"] != "[REDACTED_SECRET]" {
		t.Errorf("expected PASSWORD to be redacted, got %v", sanitized["PASSWORD"])
	}
	if sanitized["publicField"] != "public-val" {
		t.Errorf("expected publicField to remain untouched, got %v", sanitized["publicField"])
	}
}

func TestSanitizeJSON_ValidJSON(t *testing.T) {
	rawJSON := []byte(`{"service":"meshery","secret_key":"topsecret123","status":"active"}`)

	sanitizedBytes, err := SanitizeJSON(rawJSON)
	if err != nil {
		t.Fatalf("unexpected error sanitizing JSON: %v", err)
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(sanitizedBytes, &resultMap); err != nil {
		t.Fatalf("failed to unmarshal sanitized JSON: %v", err)
	}

	if resultMap["secret_key"] != "[REDACTED_SECRET]" {
		t.Errorf("expected secret_key to be redacted in JSON output, got %v", resultMap["secret_key"])
	}
	if resultMap["status"] != "active" {
		t.Errorf("expected status to remain active, got %v", resultMap["status"])
	}
}

// ==================== OMOLADE ACCEPTANCE CRITERIA TESTS ====================

// Criteria 1: Non-Mutation / Immutability Test
func TestSanitizeMap_Immutability(t *testing.T) {
	original := map[string]interface{}{
		"token": "raw-secret-token",
		"name":  "test-cluster",
	}

	_ = SanitizeMap(original)

	// Verify original caller map was not mutated in-place
	if original["token"] != "raw-secret-token" {
		t.Errorf("expected original caller map to remain immutable, but token was mutated: %v", original["token"])
	}
}

// Criteria 2: Precision Key Matching (Avoid Over-Redaction)
func TestSanitizeMap_PrecisionKeyMatching(t *testing.T) {
	input := map[string]interface{}{
		"author":     "Peaush Paul",
		"authority":  "CNCF",
		"auth_token": "secret-bearer-123",
	}

	sanitized := SanitizeMap(input)

	if sanitized["author"] != "Peaush Paul" {
		t.Errorf("over-redaction bug: author should NOT be redacted, got %v", sanitized["author"])
	}
	if sanitized["authority"] != "CNCF" {
		t.Errorf("over-redaction bug: authority should NOT be redacted, got %v", sanitized["authority"])
	}
	if sanitized["auth_token"] != "[REDACTED_SECRET]" {
		t.Errorf("expected auth_token to be redacted, got %v", sanitized["auth_token"])
	}
}

// Criteria 3: Sensitive Data in Error Paths
func TestSanitizeString_ErrorPathRedaction(t *testing.T) {
	errStr := "connection failed: auth_token=secret-xyz-789"
	sanitized := SanitizeString(errStr)

	expected := "connection failed: auth_token=[REDACTED_SECRET]"
	if sanitized != expected {
		t.Errorf("expected error string to redact token, got '%s'", sanitized)
	}
}

// Criteria 4: Malformed or Nil Input Resilience
func TestSanitizeMap_NilOrEmptyHandling(t *testing.T) {
	var nilMap map[string]interface{} = nil
	if res := SanitizeMap(nilMap); res != nil {
		t.Errorf("expected nil result for nil map input, got %v", res)
	}

	emptyJSON := []byte(``)
	resBytes, err := SanitizeJSON(emptyJSON)
	if err != nil {
		t.Fatalf("unexpected error on empty JSON: %v", err)
	}
	if len(resBytes) != 0 {
		t.Errorf("expected empty byte response for empty input, got %s", string(resBytes))
	}
}
