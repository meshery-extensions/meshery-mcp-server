package security

import (
	"encoding/json"
	"strings"
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

	if sanitized["password"] != RedactedPlaceholder {
		t.Errorf("expected password to be redacted, got %v", sanitized["password"])
	}
	if sanitized["token"] != RedactedPlaceholder {
		t.Errorf("expected token to be redacted, got %v", sanitized["token"])
	}
	if sanitized["username"] != "admin" {
		t.Errorf("expected username to be admin, got %v", sanitized["username"])
	}
}

func TestSanitizeMap_KubeconfigAndSecretData(t *testing.T) {
	input := map[string]interface{}{
		"client-key-data":            "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQ==",
		"certificate-authority-data": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t",
		"data":                       map[string]interface{}{"db_pass": "secret123"},
		"stringData":                 map[string]interface{}{"api_token": "secret456"},
		"normal_field":               "normal_value",
	}

	sanitized := SanitizeMap(input)

	if sanitized["client-key-data"] != RedactedPlaceholder {
		t.Errorf("expected client-key-data to be redacted, got %v", sanitized["client-key-data"])
	}
	if sanitized["certificate-authority-data"] != RedactedPlaceholder {
		t.Errorf("expected certificate-authority-data to be redacted, got %v", sanitized["certificate-authority-data"])
	}
	if sanitized["data"] != RedactedPlaceholder {
		t.Errorf("expected data to be redacted, got %v", sanitized["data"])
	}
	if sanitized["stringData"] != RedactedPlaceholder {
		t.Errorf("expected stringData to be redacted, got %v", sanitized["stringData"])
	}
	if sanitized["normal_field"] != "normal_value" {
		t.Errorf("expected normal_field to be untouched, got %v", sanitized["normal_field"])
	}
}

func TestSanitizeJSON_ValidJSONQuoting(t *testing.T) {
	rawJSON := []byte(`{"token":"abc123","name":"bookinfo"}`)

	sanitizedBytes, err := SanitizeJSON(rawJSON)
	if err != nil {
		t.Fatalf("unexpected error sanitizing JSON: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(sanitizedBytes, &parsed); err != nil {
		t.Fatalf("sanitized output is not valid JSON: %s (err: %v)", string(sanitizedBytes), err)
	}

	if parsed["token"] != RedactedPlaceholder {
		t.Errorf("expected token to be %s, got %v", RedactedPlaceholder, parsed["token"])
	}
	if parsed["name"] != "bookinfo" {
		t.Errorf("expected name to be bookinfo, got %v", parsed["name"])
	}
}

func TestSanitizeString_InvalidBracedJSON(t *testing.T) {
	malformedInput := `{"token": "secret-123", invalid_syntax}`

	sanitized := SanitizeString(malformedInput)
	if strings.Contains(sanitized, "secret-123") {
		t.Errorf("expected secret-123 to be redacted in malformed JSON string, got: %s", sanitized)
	}
}

func TestSanitizeMap_SecretaryNotRedacted(t *testing.T) {
	input := map[string]interface{}{
		"secretary":     "john_doe",
		"token_count":   100,
		"password_hint": "first pet name",
	}

	sanitized := SanitizeMap(input)

	if sanitized["secretary"] != "john_doe" {
		t.Errorf("expected secretary to remain unredacted, got %v", sanitized["secretary"])
	}
	if sanitized["token_count"] != 100 {
		t.Errorf("expected token_count to remain unredacted, got %v", sanitized["token_count"])
	}
}

func TestSanitizeMap_CyclicMap(t *testing.T) {
	input := map[string]interface{}{
		"name": "cyclic-test",
	}
	input["self"] = input

	sanitized := SanitizeMap(input)
	if sanitized == nil {
		t.Fatalf("expected non-nil sanitized map for cyclic input")
	}

	selfVal := sanitized["self"].(map[string]interface{})
	if selfVal["cycle"] != CircularPlaceholder {
		t.Errorf("expected cyclic reference placeholder %s, got %v", CircularPlaceholder, selfVal["cycle"])
	}
}

func TestSanitizeString_AuthorizationBearerRedaction(t *testing.T) {
	rawInput := "Authorization: Bearer my-secret-auth-token-12345"
	sanitized := SanitizeString(rawInput)

	if strings.Contains(sanitized, "my-secret-auth-token-12345") {
		t.Errorf("raw secret token leaked in sanitized string: %s", sanitized)
	}

	if !strings.Contains(sanitized, "Bearer "+RedactedPlaceholder) {
		t.Errorf("expected Bearer prefix to be preserved with placeholder, got: %s", sanitized)
	}
}

func TestSanitizeString_QuotedJSONAuthorization(t *testing.T) {
	rawInput := `{"Authorization": "Bearer secret-token-abc987", "status": "active"}`
	sanitized := SanitizeString(rawInput)

	if strings.Contains(sanitized, "secret-token-abc987") {
		t.Errorf("raw secret token leaked in quoted JSON string: %s", sanitized)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(sanitized), &parsed); err != nil {
		t.Fatalf("sanitized output is not valid JSON: %s (err: %v)", sanitized, err)
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

	if creds["kubeconfig"] != RedactedPlaceholder {
		t.Errorf("expected kubeconfig to be redacted, got %v", creds["kubeconfig"])
	}
	if creds["api_key"] != RedactedPlaceholder {
		t.Errorf("expected api_key to be redacted, got %v", creds["api_key"])
	}

	endpoints := sanitized["endpoints"].([]interface{})
	ep0 := endpoints[0].(map[string]interface{})
	if ep0["token"] != RedactedPlaceholder {
		t.Errorf("expected endpoint token to be redacted, got %v", ep0["token"])
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

	if resultMap["secret_key"] != RedactedPlaceholder {
		t.Errorf("expected secret_key to be redacted, got %v", resultMap["secret_key"])
	}
}
