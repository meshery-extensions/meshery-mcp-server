package meshery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient_Defaults(t *testing.T) {
	t.Setenv("MESHERY_SERVER_URL", "")
	t.Setenv("MESHERY_API_TOKEN", "")

	client := NewClient()

	if client.BaseURL != "http://localhost:9081" {
		t.Fatalf("expected default URL, got %s", client.BaseURL)
	}

	if client.HTTP == nil {
		t.Fatal("expected HTTP client to be initialized")
	}
}

func TestNewClient_Environment(t *testing.T) {
	t.Setenv("MESHERY_SERVER_URL", "http://example.com/")
	t.Setenv("MESHERY_API_TOKEN", "test-token")

	client := NewClient()

	if client.BaseURL != "http://example.com" {
		t.Fatalf("expected trimmed URL, got %s", client.BaseURL)
	}

	if client.APIToken != "test-token" {
		t.Fatalf("expected API token, got %s", client.APIToken)
	}
}

func TestClientDo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization header")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := &Client{
		BaseURL:  server.URL,
		APIToken: "test-token",
		HTTP:     server.Client(),
	}

	var result struct {
		Status string `json:"status"`
	}

	err := client.Do(
		context.Background(),
		http.MethodGet,
		"/api/test",
		nil,
		&result,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Status != "ok" {
		t.Fatalf("expected status ok, got %s", result.Status)
	}
}

func TestClientDo_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &Client{
		BaseURL: server.URL,
		HTTP:    server.Client(),
	}

	err := client.Do(
		context.Background(),
		http.MethodGet,
		"/api/test",
		nil,
		nil,
	)

	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}
