package meshery_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
)

func TestIntegration_ListDesigns(t *testing.T) {
	// 1. Establish an isolated temporary workspace for the credentials file
	tmpDir, err := os.MkdirTemp("", "meshery-mcp-test")
	if err != nil {
		t.Fatalf("Failed to establish a temporary workspace: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mockAuthPath := filepath.Join(tmpDir, "auth.json")
	mockAuthData := map[string]string{
		"token":            "mock-integration-test-token",
		"meshery-provider": "None",
	}
	authBytes, _ := json.Marshal(mockAuthData)
	_ = os.WriteFile(mockAuthPath, authBytes, 0644)

	// 2. Spin up an internal mock server that acts exactly like a healthy Meshery endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify that the client engine successfully injected the cookie
		cookie, err := r.Cookie("meshery-token")
		if err != nil {
			t.Errorf("Authentication failure: meshery-token cookie is missing from the request")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if cookie.Value != "mock-integration-test-token" {
			t.Errorf("Data corruption: expected token 'mock-integration-test-token', got '%s'", cookie.Value)
			w.WriteHeader(http.StatusForbidden)
			return
		}

		// Respond with a mock successful JSON payload
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "success", "components": []}`))
	}))
	defer mockServer.Close()

	// 3. Point your client engine configuration at the fast internal test listener
	cfg := meshery.ClientConfig{
		ServerURL:  mockServer.URL,
		ConfigPath: mockAuthPath,
	}

	client, err := meshery.NewAuthenticatedClient(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize authenticated integration client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 4. Fire the test request
	req, _ := http.NewRequestWithContext(ctx, "GET", cfg.ServerURL+"/api/system/meshsync/resources", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to reach the mock test server: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Unexpected HTTP status code: %d. Response payload: %s", resp.StatusCode, string(body))
	}
}
