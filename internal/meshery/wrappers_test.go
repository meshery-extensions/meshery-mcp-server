package meshery

import (
	"context"
	"strings"
	"testing"
)

// wrapperCase asserts the route, pagination, cookie auth and absence of
// credential leakage for one read-only wrapper.
func wrapperCase(t *testing.T, name, body, wantPath, wantQuery string, call func(*Client) error) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		srv, mock := newAPIMock(t, 200, body)
		defer srv.Close()
		c := NewClient(srv.URL, "secret-token", WithAuthMode(AuthCookie), WithProvider("cloud"))

		if err := call(c); err != nil {
			t.Fatalf("%s failed: %v", name, err)
		}
		reqs := mock.calls()
		if len(reqs) != 1 {
			t.Fatalf("%s made %d requests, want 1", name, len(reqs))
		}
		req := reqs[0]
		if req.path != wantPath {
			t.Errorf("%s path = %q, want %q", name, req.path, wantPath)
		}
		if !strings.Contains(req.query, wantQuery) {
			t.Errorf("%s query = %q, want %q", name, req.query, wantQuery)
		}
		if !strings.Contains(req.cookie, "token=secret-token") || !strings.Contains(req.cookie, "meshery-provider=cloud") {
			t.Errorf("%s cookie = %q, want token and provider", name, req.cookie)
		}
		if req.auth != "" {
			t.Errorf("%s sent Authorization header: %q", name, req.auth)
		}
	})
}

func TestEnvironmentWrappers(t *testing.T) {
	page := `{"environments":[{"id":"e1","name":"default","organizationId":"o1","purpose":"user"}],"page":0,"pageSize":10,"totalCount":1}`
	wrapperCase(t, "list", page, "/api/environments", "orgId=o1", func(c *Client) error {
		got, err := c.ListEnvironments(context.Background(), "o1", ListOptions{PageSize: 10})
		if err == nil && (len(got.Environments) != 1 || got.Environments[0].Name != "default" || got.TotalCount != 1) {
			t.Errorf("unexpected page: %+v", got)
		}
		return err
	})
	wrapperCase(t, "get", `{"id":"e1","name":"default"}`, "/api/environments/e1", "", func(c *Client) error {
		got, err := c.GetEnvironment(context.Background(), "e1")
		if err == nil && got.ID != "e1" {
			t.Errorf("unexpected environment: %+v", got)
		}
		return err
	})
}

func TestWorkspaceWrappers(t *testing.T) {
	wrapperCase(t, "list", `{"workspaces":[{"id":"w1","name":"team"}],"page":1,"pageSize":25,"totalCount":7}`,
		"/api/workspaces", "orgId=o1", func(c *Client) error {
			got, err := c.ListWorkspaces(context.Background(), "o1", ListOptions{Page: 1, PageSize: 25})
			if err == nil && (len(got.Workspaces) != 1 || got.Page != 1 || got.TotalCount != 7) {
				t.Errorf("unexpected page: %+v", got)
			}
			return err
		})
	wrapperCase(t, "get", `{"id":"w1","name":"team"}`, "/api/workspaces/w1", "", func(c *Client) error {
		got, err := c.GetWorkspace(context.Background(), "w1")
		if err == nil && got.Name != "team" {
			t.Errorf("unexpected workspace: %+v", got)
		}
		return err
	})
}

func TestDesignWrappers(t *testing.T) {
	wrapperCase(t, "list", `{"patterns":[{"id":"d1","name":"bookinfo","patternFile":"kind: Service"}],"page":0,"pageSize":10,"totalCount":1}`,
		"/api/pattern", "pagesize=10", func(c *Client) error {
			got, err := c.ListDesigns(context.Background(), DesignListOptions{ListOptions: ListOptions{PageSize: 10}})
			if err == nil && (len(got.Patterns) != 1 || got.Patterns[0].Name != "bookinfo") {
				t.Errorf("unexpected page: %+v", got)
			}
			return err
		})
	wrapperCase(t, "get", `{"id":"d1","name":"bookinfo","patternFile":"kind: Service"}`,
		"/api/pattern/d1", "", func(c *Client) error {
			got, err := c.GetDesign(context.Background(), "d1")
			if err == nil && !strings.Contains(string(got.PatternFile), "Service") {
				t.Errorf("unexpected design: %+v", got)
			}
			return err
		})
}

func TestConnectionWrappers(t *testing.T) {
	wrapperCase(t, "list", `{"connections":[{"id":"c1","name":"k8s","kind":"kubernetes"}],"page":0,"pageSize":10,"totalCount":1}`,
		"/api/integrations/connections", "pageSize=10", func(c *Client) error {
			got, err := c.ListConnections(context.Background(), ConnectionListOptions{
				ListOptions: ListOptions{PageSize: 10}, Kinds: []string{"kubernetes"},
			})
			if err == nil && (len(got.Connections) != 1 || got.Connections[0].Kind != "kubernetes") {
				t.Errorf("unexpected page: %+v", got)
			}
			return err
		})
	wrapperCase(t, "get", `{"id":"c1","name":"k8s"}`, "/api/integrations/connections/c1", "", func(c *Client) error {
		got, err := c.GetConnection(context.Background(), "c1")
		if err == nil && got.ID != "c1" {
			t.Errorf("unexpected connection: %+v", got)
		}
		return err
	})
}

func TestRegistryWrappers(t *testing.T) {
	wrapperCase(t, "models", `{"page":0,"pageSize":10,"totalCount":1,"models":[{"id":"m1","name":"nginx"}]}`,
		"/api/registry/models", "pageSize=10", func(c *Client) error {
			got, err := c.ListModels(context.Background(), RegistryListOptions{Version: "v1.0.0"})
			if err == nil && (len(got.Entries) != 1 || got.TotalCount != 1) {
				t.Errorf("unexpected page: %+v", got)
			}
			return err
		})
	wrapperCase(t, "components", `{"page":0,"pageSize":10,"totalCount":1,"components":[{"id":"comp1","name":"nginx"}]}`,
		"/api/registry/components", "version=v1.0.0", func(c *Client) error {
			got, err := c.ListComponents(context.Background(), RegistryListOptions{Version: "v1.0.0"})
			if err == nil && len(got.Entries) != 1 {
				t.Errorf("unexpected page: %+v", got)
			}
			return err
		})
}

func TestPerformanceWrappers(t *testing.T) {
	wrapperCase(t, "profiles", `{"profiles":[{"id":"p1","name":"smoke"}],"page":0,"pageSize":10,"totalCount":1}`,
		"/api/performance/profiles", "pagesize=10", func(c *Client) error {
			got, err := c.ListPerformanceProfiles(context.Background(), ListOptions{})
			if err == nil && (len(got.Profiles) != 1 || got.Profiles[0].Name != "smoke") {
				t.Errorf("unexpected page: %+v", got)
			}
			return err
		})
	wrapperCase(t, "results", `{"results":[{"id":"r1"}],"page":0,"pageSize":10,"totalCount":1}`,
		"/api/performance/results", "page=0", func(c *Client) error {
			got, err := c.ListPerformanceResults(context.Background(), ListOptions{})
			if err == nil && len(got.Results) != 1 {
				t.Errorf("unexpected page: %+v", got)
			}
			return err
		})
}

func TestWrappersClampPageSizeAndAcceptEmptyResults(t *testing.T) {
	srv, mock := newAPIMock(t, 200, `{"patterns":[],"page":0,"pageSize":100,"totalCount":0}`)
	defer srv.Close()
	c := newTestClient(t, srv, "")

	page, err := c.ListDesigns(context.Background(), DesignListOptions{
		ListOptions: ListOptions{Page: -5, PageSize: 500},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Patterns) != 0 || page.TotalCount != 0 {
		t.Fatalf("expected empty page, got %+v", page)
	}
	q := mock.calls()[0].query
	if !strings.Contains(q, "page=0") || !strings.Contains(q, "pagesize=100") {
		t.Fatalf("pagination was not clamped: %q", q)
	}
	if mock.calls()[0].cookie != "" {
		t.Fatalf("sent credentials without a token: %q", mock.calls()[0].cookie)
	}
}

func TestWrappersMalformedJSON(t *testing.T) {
	srv, _ := newAPIMock(t, 200, `{not-json`)
	defer srv.Close()
	c := newTestClient(t, srv, "token")
	if _, err := c.ListWorkspaces(context.Background(), "", ListOptions{}); err == nil || !strings.Contains(err.Error(), "decoding") {
		t.Fatalf("expected decode error, got %v", err)
	}
}
