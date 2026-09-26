package security

import (
	"testing"
)

func BenchmarkSanitizeMap(b *testing.B) {
	input := map[string]interface{}{
		"username": "admin",
		"password": "supersecretpassword123",
		"token":    "bearer-token-xyz-789",
		"cluster":  "k8s-prod-cluster",
		"metadata": map[string]interface{}{
			"kubeconfig": "apiVersion: v1...",
			"api_key":    "key-12345",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SanitizeMap(input)
	}
}

func BenchmarkSanitizeJSON(b *testing.B) {
	rawJSON := []byte(`{"service":"meshery","secret_key":"topsecret123","status":"active","metadata":{"token":"secret-token"}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = SanitizeJSON(rawJSON)
	}
}
