package mesherytest_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/meshery-extensions/meshery-mcp-server/internal/mesherytest"
)

// getWith sends one authenticated-looking request with the cookies it is given,
// so a test can send a correct token alongside a wrong provider. The recorder
// wraps the mux, so the request is recorded whether or not the guard admits it.
func getWith(t *testing.T, s *mesherytest.Server, path, token, provider string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, s.URL()+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "token", Value: token})
	}
	if provider != "" {
		req.AddCookie(&http.Cookie{Name: "meshery-provider", Value: provider})
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
}

// TestAuthenticatedJudgesTokenAndProviderSeparately is the disagreeing fixture
// for the two halves of AssertAuthenticated.
//
// Every other test sends both credentials correctly, so the two checks always
// agree and either one could be deleted with the suite still green. Meshery
// reads them from different places and fails differently: a wrong token is not
// a session, and a wrong provider selects a different backend that has never
// heard of the session. Each half is exercised here with the other half right.
func TestAuthenticatedJudgesTokenAndProviderSeparately(t *testing.T) {
	for _, tc := range []struct {
		name       string
		token      string
		provider   string
		wantInMsg  string
		shouldFail bool
	}{
		{"both correct", "", "", "", false},
		{"right token, wrong provider", "", "SomeoneElse", "provider", true},
		{"right provider, wrong token", "not-the-session", "", "token", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := mesherytest.New(t)
			token, provider := s.Token, s.Provider
			if tc.token != "" {
				token = tc.token
			}
			if tc.provider != "" {
				provider = tc.provider
			}
			getWith(t, s, "/api/system/kubernetes/contexts", token, provider)

			failed, msg := (&recorder{}).run(func(tt mesherytest.T) {
				s.AssertAuthenticated(tt)
			})
			if failed != tc.shouldFail {
				t.Fatalf("AssertAuthenticated failed = %v, want %v (%s)", failed, tc.shouldFail, msg)
			}
			if tc.shouldFail && !strings.Contains(strings.ToLower(msg), tc.wantInMsg) {
				t.Errorf("the message should name the %s that was wrong, got %q", tc.wantInMsg, msg)
			}
		})
	}
}

// TestProviderIsAcceptedFromAnyOfItsThreeChannels covers the other half of the
// provider rule. Meshery takes it from the cookie, else a header of the same
// name, else the query parameter, so an assertion insisting on the cookie would
// reject two ways a real client legitimately sends it.
func TestProviderIsAcceptedFromAnyOfItsThreeChannels(t *testing.T) {
	for _, via := range []string{"header", "query"} {
		t.Run(via, func(t *testing.T) {
			s := mesherytest.New(t)
			url := s.URL() + "/api/system/kubernetes/contexts"
			if via == "query" {
				url += "?provider=" + s.Provider
			}
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.AddCookie(&http.Cookie{Name: "token", Value: s.Token})
			if via == "header" {
				req.Header.Set("meshery-provider", s.Provider)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			_ = resp.Body.Close()

			if failed, msg := (&recorder{}).run(func(tt mesherytest.T) {
				s.AssertAuthenticated(tt)
			}); failed {
				t.Errorf("a provider sent by %s should satisfy the assertion: %s", via, msg)
			}
		})
	}
}
