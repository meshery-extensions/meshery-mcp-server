package meshery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type captureTransport struct {
	capturedReq *http.Request
}

func (ct *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ct.capturedReq = req
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/json")
	rec.WriteHeader(http.StatusOK)
	_, _ = rec.WriteString(`{"totalCount": 1, "patterns": []}`)
	return rec.Result(), nil
}

func TestMesheryClient_ListDesigns_QueryStringAndDualCookies(t *testing.T) {
	var capturedPage, capturedPageSize, capturedSearch, capturedAuth string
	var capturedTokenCookie, capturedProviderCookie *http.Cookie

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPage = r.URL.Query().Get("page")
		capturedPageSize = r.URL.Query().Get("pagesize")
		capturedSearch = r.URL.Query().Get("search")
		capturedAuth = r.Header.Get("Authorization")

		capturedTokenCookie, _ = r.Cookie("token")
		capturedProviderCookie, _ = r.Cookie("meshery-provider")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Meshery Server sends totalCount (PascalCase camelTag matching PatternsAPIResponse)
		_, _ = w.Write([]byte(`{"totalCount": 15, "patterns": [{"id": "p-1", "name": "K8s-Pattern"}]}`))
	}))
	defer ts.Close()

	// httptest server runs on 127.0.0.1 loopback, which passes loopback credential guard
	client := NewClient(ts.URL, "test-bearer-token", "Meshery")

	designs, totalCount, err := client.ListDesigns(context.Background(), 2, 25, "kubernetes")
	if err != nil {
		t.Fatalf("unexpected error querying ListDesigns: %v", err)
	}

	if totalCount != 15 {
		t.Errorf("expected totalCount 15 from totalCount tag, got %d", totalCount)
	}

	if len(designs) != 1 {
		t.Errorf("expected 1 design, got %d", len(designs))
	}

	if capturedPage != "2" {
		t.Errorf("expected query page=2, got %s", capturedPage)
	}

	if capturedPageSize != "25" {
		t.Errorf("expected query pagesize=25, got %s", capturedPageSize)
	}

	if capturedSearch != "kubernetes" {
		t.Errorf("expected query search=kubernetes, got %s", capturedSearch)
	}

	if capturedAuth != "Bearer test-bearer-token" {
		t.Errorf("expected Authorization header Bearer test-bearer-token, got %s", capturedAuth)
	}

	if capturedTokenCookie == nil || capturedTokenCookie.Value != "test-bearer-token" {
		t.Errorf("expected token cookie test-bearer-token, got %v", capturedTokenCookie)
	}

	if capturedProviderCookie == nil || capturedProviderCookie.Value != "Meshery" {
		t.Errorf("expected meshery-provider cookie Meshery, got %v", capturedProviderCookie)
	}
}

func TestMesheryClient_ListDesigns_HTTPCleartextCredentialWithholding(t *testing.T) {
	ct := &captureTransport{}
	clientInterface := NewClient("http://remote-unencrypted-server.internal", "secret-token-123", "Meshery")

	if mc, ok := clientInterface.(*mesheryClient); ok {
		mc.httpClient.Transport = ct
	} else {
		t.Fatalf("failed to cast Client interface to concrete *mesheryClient")
	}

	_, _, err := clientInterface.ListDesigns(context.Background(), 0, 10, "")
	if err != nil {
		t.Fatalf("unexpected error executing ListDesigns with mock capture transport: %v", err)
	}

	req := ct.capturedReq
	if req == nil {
		t.Fatalf("expected request to be captured by mock transport, got nil")
	}

	if auth := req.Header.Get("Authorization"); auth != "" {
		t.Errorf("expected Authorization header to be withheld over cleartext HTTP, got: %s", auth)
	}

	if cookie, err := req.Cookie("token"); err != http.ErrNoCookie {
		t.Errorf("expected token cookie to be withheld over cleartext HTTP, got: %v", cookie)
	}

	if cookie, err := req.Cookie("meshery-provider"); err != http.ErrNoCookie {
		t.Errorf("expected meshery-provider cookie to be withheld over cleartext HTTP, got: %v", cookie)
	}
}

func TestMesheryClient_ListDesigns_BoundedErrorBodyReading(t *testing.T) {
	// Create an oversized error response body (100KB)
	largeErrorBody := strings.Repeat("error detail block ", 5000)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(largeErrorBody))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	_, _, err := client.ListDesigns(context.Background(), 0, 10, "")
	if err == nil {
		t.Fatalf("expected error from internal server error response, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "meshery API returned status 500") {
		t.Errorf("expected status 500 in error message, got: %s", errMsg)
	}

	// Verify error body was bounded (less than full 100KB)
	if len(errMsg) > 70*1024 {
		t.Errorf("error message exceeds 70KB limit, read body was not properly bounded: len=%d", len(errMsg))
	}
}

func TestMesheryClient_Ping(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/system/version" {
			t.Errorf("expected path /api/system/version, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"build": "v1.0.0", "commit": "abcdef"}`))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	ver, err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on Ping: %v", err)
	}
	if ver["build"] != "v1.0.0" {
		t.Errorf("expected build v1.0.0, got %v", ver["build"])
	}
}

func TestMesheryClient_GetEnvironments_OrgIDQuery(t *testing.T) {
	var capturedOrgID string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedOrgID = r.URL.Query().Get("orgId")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"totalCount": 2, "environments": [{"id": "e-1", "name": "staging"}, {"id": "e-2", "name": "prod"}]}`))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	envs, total, err := client.GetEnvironments(context.Background(), "org-123", 0, 10)
	if err != nil {
		t.Fatalf("unexpected error on GetEnvironments: %v", err)
	}
	if capturedOrgID != "org-123" {
		t.Errorf("expected query orgId=org-123, got %s", capturedOrgID)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(envs) != 2 {
		t.Errorf("expected 2 environments, got %d", len(envs))
	}
}

func TestMesheryClient_GetConnections(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/system/meshsync/connections" {
			t.Errorf("expected path /api/system/meshsync/connections, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"totalCount": 1, "connections": [{"id": "c-1", "status": "connected"}]}`))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	conns, total, err := client.GetConnections(context.Background(), 0, 10)
	if err != nil {
		t.Fatalf("unexpected error on GetConnections: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(conns) != 1 {
		t.Errorf("expected 1 connection, got %d", len(conns))
	}
}

func TestMesheryClient_GetAdapters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/system/adapters" {
			t.Errorf("expected path /api/system/adapters, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"name": "meshery-istio", "location": "localhost:10000"}]`))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	adapters, err := client.GetAdapters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on GetAdapters: %v", err)
	}
	if len(adapters) != 1 {
		t.Errorf("expected 1 adapter, got %d", len(adapters))
	}
}

