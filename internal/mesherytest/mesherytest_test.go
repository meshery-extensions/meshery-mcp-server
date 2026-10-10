package mesherytest_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/meshery-extensions/meshery-mcp-server/internal/mesherytest"
)

const resourcesPath = "/api/system/meshsync/resources"

// naiveGet is a client written the way a Meshery client is usually written the
// first time: a bearer token, no cluster filter, one-based pages. Every one of
// these choices is wrong against a real Meshery, and none of them produce an
// error. The tests below use it to show what the fake catches.
func naiveGet(t *testing.T, base, path, query string) (int, []byte) {
	t.Helper()
	u := base + path
	if query != "" {
		u += "?" + query
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+mesherytest.DefaultToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body
}

func authedGet(t *testing.T, s *mesherytest.Server, path, query string) map[string]any {
	t.Helper()
	u := s.URL() + path
	if query != "" {
		u += "?" + query
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "token", Value: s.Token})
	req.AddCookie(&http.Cookie{Name: "meshery-provider", Value: s.Provider})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decoding %s: %v", path, err)
	}
	return out
}

// TestBearerAuthLandsOnLoginPage is the first bug the fake reproduces. A bearer
// token is not a Meshery session, and the failure is not a 401: the request is
// redirected, the redirect is followed, and the client parses an HTML login
// page. A mock that checks the Authorization header would call this a pass.
//
// Note where the redirect goes. With no token cookie, GetSession returns
// ErrEmptySession, which AuthMiddleware excludes from HandleUnAuthenticated, so
// the request lands in LoginHandler and is sent to the remote provider's own
// login page rather than to anything on Meshery.
func TestBearerAuthLandsOnLoginPage(t *testing.T) {
	s := mesherytest.New(t)

	status, body := naiveGet(t, s.URL(), "/api/system/meshsync/resources", "")

	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200: Meshery redirects rather than refusing, and the redirect target succeeds", status)
	}
	if !strings.Contains(string(body), "Sign in") {
		t.Fatalf("expected the login page, got: %s", body)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err == nil {
		t.Fatal("expected the HTML login page to fail JSON decoding")
	}
}

// TestLocalProviderAcceptsAnything is why the bug above is latent rather than
// obvious. DefaultLocalProvider.GetSession returns nil unconditionally, so a
// client with no working authentication passes every test against a locally
// started Meshery and fails the first time it meets a remote provider.
func TestLocalProviderAcceptsAnything(t *testing.T) {
	s := mesherytest.New(t, mesherytest.WithLocalProvider())

	status, body := naiveGet(t, s.URL(), "/api/system/kubernetes/contexts", "")

	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if !strings.Contains(string(body), "minikube") {
		t.Fatalf("a local provider should have served the data anyway, got: %s", body)
	}
}

// TestProviderHasThreeChannels covers the asymmetry between the two credentials.
// The token is read from its cookie and nowhere else, but the provider falls
// back to a header of the same name and then to ?provider=, so a client that
// sends the provider either of those other ways is not broken and the fake must
// not pretend otherwise.
func TestProviderHasThreeChannels(t *testing.T) {
	const path = "/api/system/kubernetes/contexts"

	for _, tc := range []struct {
		channel string
		apply   func(*http.Request, *mesherytest.Server)
	}{
		{"cookie", func(r *http.Request, s *mesherytest.Server) {
			r.AddCookie(&http.Cookie{Name: "meshery-provider", Value: s.Provider})
		}},
		{"header", func(r *http.Request, s *mesherytest.Server) {
			r.Header.Set("meshery-provider", s.Provider)
		}},
		{"query", func(r *http.Request, s *mesherytest.Server) {
			q := r.URL.Query()
			q.Set("provider", s.Provider)
			r.URL.RawQuery = q.Encode()
		}},
	} {
		t.Run(tc.channel, func(t *testing.T) {
			s := mesherytest.New(t)
			req, _ := http.NewRequest(http.MethodGet, s.URL()+path, nil)
			req.AddCookie(&http.Cookie{Name: "token", Value: s.Token})
			tc.apply(req, s)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(body), "minikube") {
				t.Fatalf("provider via %s was not accepted: %s", tc.channel, body)
			}

			if _, got := s.Requests()[0].Provider(); got != tc.channel {
				t.Errorf("recorded channel = %q, want %q", got, tc.channel)
			}
			s.AssertAuthenticated(t)
		})
	}
}

// TestTokenIsCookieOnly is the other half of that asymmetry: the provider has
// three channels, the session has one.
func TestTokenIsCookieOnly(t *testing.T) {
	s := mesherytest.New(t)

	req, _ := http.NewRequest(http.MethodGet, s.URL()+"/api/system/kubernetes/contexts?token="+s.Token, nil)
	req.Header.Set("token", s.Token)
	req.AddCookie(&http.Cookie{Name: "meshery-provider", Value: s.Provider})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Sign in") {
		t.Fatalf("a token sent as a header or query param is not a session, got: %s", body)
	}
}

// TestInvalidTokenTakesTheOtherPath separates the two rejection routes. A token
// cookie that is present but wrong does reach HandleUnAuthenticated, which stays
// on Meshery and redirects to /auth/login or /provider depending on whether the
// provider cookie is there.
func TestInvalidTokenTakesTheOtherPath(t *testing.T) {
	s := mesherytest.New(t)

	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	// A provider is selected throughout, so the gate is past and what is left is
	// the session decision.
	req, _ := http.NewRequest(http.MethodGet, s.URL()+"/api/pattern", nil)
	req.AddCookie(&http.Cookie{Name: "meshery-provider", Value: s.Provider})
	req.AddCookie(&http.Cookie{Name: "token", Value: "stale-or-expired"})
	resp, err := noRedirect.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got := resp.Header.Get("Location"); got != "/auth/login" {
		t.Errorf("a present but invalid token redirected to %q, want /auth/login", got)
	}

	// Provider selected and no token at all: the provider's own login page,
	// which is off-host on a real deployment.
	req, _ = http.NewRequest(http.MethodGet, s.URL()+"/api/pattern", nil)
	req.AddCookie(&http.Cookie{Name: "meshery-provider", Value: s.Provider})
	resp, err = noRedirect.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got := resp.Header.Get("Location"); got != mesherytest.RemoteLoginPath {
		t.Errorf("with no token: redirected to %q, want %q", got, mesherytest.RemoteLoginPath)
	}
}

// TestProviderGateAnswersBeforeTheSession covers the first thing an
// unauthenticated client actually meets, which is not the session check. With no
// provider selected, AuthMiddleware redirects to /provider carrying the original
// path base64-encoded in ref, and it does that for every method rather than only
// for GET.
//
// Verified against a running Meshery: an unauthenticated GET, POST and DELETE
// all came back 302 to /provider?ref=..., before any session logic ran.
func TestProviderGateAnswersBeforeTheSession(t *testing.T) {
	s := mesherytest.New(t)
	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	want := "/provider?ref=" + base64.RawURLEncoding.EncodeToString([]byte("/api/pattern"))
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodPut} {
		req, _ := http.NewRequest(method, s.URL()+"/api/pattern", nil)
		resp, err := noRedirect.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusFound {
			t.Errorf("%s with no provider: status = %d, want 302", method, resp.StatusCode)
			continue
		}
		if got := resp.Header.Get("Location"); got != want {
			t.Errorf("%s: Location = %q, want %q", method, got, want)
		}
	}
}

// TestNonGetWithAProviderButNoTokenIs404 covers the step past the gate. A
// missing token is ErrEmptySession, which AuthMiddleware excludes from
// HandleUnAuthenticated, so it lands in LoginHandler and a non-GET gets a bare
// 404 that reads like a missing route rather than a missing session.
func TestNonGetWithAProviderButNoTokenIs404(t *testing.T) {
	s := mesherytest.New(t)

	req, _ := http.NewRequest(http.MethodPost, s.URL()+"/api/pattern", nil)
	req.AddCookie(&http.Cookie{Name: "meshery-provider", Value: s.Provider})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// TestMissingClusterFilterReturnsNothing is the second bug. Meshery filters
// with cluster_id IN (?), so no filter is an empty IN clause: 200, an empty
// list, and a model that reports the cluster is empty.
func TestMissingClusterFilterReturnsNothing(t *testing.T) {
	s := mesherytest.New(t)

	out := authedGet(t, s, "/api/system/meshsync/resources", "")
	if n := out["totalCount"].(float64); n != 0 {
		t.Fatalf("totalCount = %v, want 0 without a cluster filter", n)
	}

	out = authedGet(t, s, "/api/system/meshsync/resources", `clusterIds=["`+s.Data().ClusterID()+`"]`)
	if want := float64(len(s.Data().Resources)); out["totalCount"].(float64) != want {
		t.Fatalf("totalCount = %v, want %v with the filter", out["totalCount"], want)
	}
}

// TestBareClusterIDIsRejected covers the near miss: the parameter is present and
// looks right, but the handler json.Unmarshals it into a []string and answers
// 400 when that fails. Unlike the absent case above, this one is loud, and the
// distinction is worth reproducing exactly rather than collapsing both into an
// empty result.
func TestBareClusterIDIsRejected(t *testing.T) {
	s := mesherytest.New(t)

	req, _ := http.NewRequest(http.MethodGet,
		s.URL()+"/api/system/meshsync/resources?clusterIds="+s.Data().ClusterID(), nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: s.Token})
	req.AddCookie(&http.Cookie{Name: "meshery-provider", Value: s.Provider})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: a bare id is not a JSON array", resp.StatusCode)
	}
}

// TestSummaryUsesADifferentSpelling covers the trap between the two sibling
// endpoints: resources takes a JSON clusterIds array, summary takes a repeated
// singular clusterId and answers 400 without one. Reusing the first spelling on
// the second endpoint fails outright.
func TestSummaryUsesADifferentSpelling(t *testing.T) {
	s := mesherytest.New(t)
	const path = "/api/system/meshsync/resources/summary"

	req, _ := http.NewRequest(http.MethodGet, s.URL()+path+`?clusterIds=["`+s.Data().ClusterID()+`"]`, nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: s.Token})
	req.AddCookie(&http.Cookie{Name: "meshery-provider", Value: s.Provider})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 when the singular clusterId is absent", resp.StatusCode)
	}

	out := authedGet(t, s, path, "clusterId="+s.Data().ClusterID())
	if out["kinds"] == nil {
		t.Fatalf("expected per-kind counts, got %v", out)
	}
}

// TestSummaryClusterIDIsAPresenceCheck covers the shape of the summary guard.
// It tests that the clusterId key is present, not that it holds anything useful,
// so an empty value or the literal "all" that the UI sends both sail past it.
//
// What comes back is the trap. The value is then used as an ordinary cluster id
// in an IN clause, so neither matches a row and every field of the summary is
// null under a 200. Measured against a live server: clusterId=all and
// clusterId= both return {"kinds":null,"namespaces":null,"labels":null}. "all"
// is not a wildcard, it is a value nothing is named.
func TestSummaryClusterIDIsAPresenceCheck(t *testing.T) {
	s := mesherytest.New(t)
	const path = "/api/system/meshsync/resources/summary"

	// The real cluster id is the control: the same endpoint does answer. It runs
	// first because the assertion below judges the last request to this path.
	if out := authedGet(t, s, path, "clusterId="+s.Data().ClusterID()); out["kinds"] == nil {
		t.Errorf("the seeded cluster should have a census: %v", out)
	}

	for _, q := range []string{"clusterId=", "clusterId=all"} {
		out := authedGet(t, s, path, q)
		for _, key := range []string{"kinds", "namespaces", "labels"} {
			if out[key] != nil {
				t.Errorf("%s: %s should be null, a request naming no cluster matches nothing: %v", q, key, out)
			}
		}
	}

	// And the assertion still objects, because neither value names the cluster.
	failed, _ := (&recorder{}).run(func(tt mesherytest.T) {
		s.AssertClusterScoped(tt, path, s.Data().ClusterID())
	})
	if !failed {
		t.Error("clusterId=all should not satisfy a cluster-scoping assertion")
	}
}

// TestPageOneSkipsTheFirstPage is the third bug. Pagination is zero-based on
// both of Meshery's offset paths, so a client that opens at page 1 misses the
// first page of every list. With one seeded cluster, page 1 is empty.
func TestPageOneSkipsTheFirstPage(t *testing.T) {
	s := mesherytest.New(t)

	out := authedGet(t, s, "/api/system/kubernetes/contexts", "page=1&pageSize=25")
	if ctxs := out["contexts"].([]any); len(ctxs) != 0 {
		t.Fatalf("page 1 returned %d contexts; zero-based paging means the first page is page 0", len(ctxs))
	}

	out = authedGet(t, s, "/api/system/kubernetes/contexts", "page=0&pageSize=25")
	if ctxs := out["contexts"].([]any); len(ctxs) != 1 {
		t.Fatalf("page 0 returned %d contexts, want 1", len(ctxs))
	}
}

// TestPageSizeSpellingIsPerEndpoint is the trap that a fake accepting both
// spellings everywhere would hide. Meshery is not consistent: most handlers read
// only the lowercase pagesize, and there the camelCase pageSize is ignored and
// that endpoint's default quietly applies. Two paths read pageSize first and fall back
// to pagesize. Nothing errors either way.
func TestPageSizeSpellingIsPerEndpoint(t *testing.T) {
	s := mesherytest.New(t)

	// /api/pattern is served by GetMesheryPatternsHandler, which reads
	// q.Get("pagesize") and nothing else.
	out := authedGet(t, s, "/api/pattern", "pagesize=1")
	if n := len(out["patterns"].([]any)); n != 1 {
		t.Errorf("lowercase pagesize=1 returned %d designs, want 1", n)
	}
	out = authedGet(t, s, "/api/pattern", "pageSize=1")
	if n := len(out["patterns"].([]any)); n != 10 {
		t.Errorf("camelCase pageSize=1 returned %d designs, want this endpoint's default of 10: it ignores that spelling", n)
	}

	// /api/system/meshsync/resources goes through getPaginationParams, which
	// reads pageSize first and falls back to pagesize, so both work there.
	cluster := `clusterIds=["` + s.Data().ClusterID() + `"]`
	for _, spelling := range []string{"pageSize=1", "pagesize=1"} {
		out = authedGet(t, s, "/api/system/meshsync/resources", cluster+"&"+spelling)
		if n := len(out["resources"].([]any)); n != 1 {
			t.Errorf("%s returned %d resources, want 1: this endpoint reads both spellings", spelling, n)
		}
	}
}

// TestAssertPageSizeSpellingCatchesTheWrongOne checks the assertion fires when a
// client sends a spelling the endpoint does not read.
func TestAssertPageSizeSpellingCatchesTheWrongOne(t *testing.T) {
	s := mesherytest.New(t)
	authedGet(t, s, "/api/pattern", "pageSize=1")

	failed, msg := (&recorder{}).run(func(tt mesherytest.T) {
		s.AssertPageSizeSpelling(tt, "/api/pattern")
	})
	if !failed {
		t.Fatal("camelCase pageSize on a lowercase-only endpoint should have been flagged")
	}
	if !strings.Contains(msg, "pagesize") {
		t.Errorf("the failure should name the spelling the endpoint reads, got: %s", msg)
	}

	// And it passes on the right one.
	s2 := mesherytest.New(t)
	authedGet(t, s2, "/api/pattern", "pagesize=1")
	if failed, msg := (&recorder{}).run(func(tt mesherytest.T) {
		s2.AssertPageSizeSpelling(tt, "/api/pattern")
	}); failed {
		t.Errorf("lowercase pagesize should be accepted here, got: %s", msg)
	}
}

// TestDefaultPageSizeIsPerEndpoint pins the defaults measured against a running
// Meshery. They are not uniform, and they do not follow from the spelling: the
// connections endpoint reads the camelCase spelling and still defaults to 10,
// while meshsync resources reads it and defaults to 25. A client that assumes
// one number everywhere mis-pages against four of these six.
func TestDefaultPageSizeIsPerEndpoint(t *testing.T) {
	s := mesherytest.New(t)

	for _, tc := range []struct {
		path  string
		query string
		key   string
		want  float64
	}{
		{"/api/pattern", "", "patterns", 10},
		{"/api/system/kubernetes/contexts", "", "contexts", 10},
		{"/api/integrations/connections", "", "connections", 10},
		{"/api/system/meshsync/resources", `clusterIds=["` + s.Data().ClusterID() + `"]`, "resources", 25},
	} {
		out := authedGet(t, s, tc.path, tc.query)
		if got := out["pageSize"].(float64); got != tc.want {
			t.Errorf("%s: default pageSize = %v, want %v", tc.path, got, tc.want)
		}
	}
}

// TestPageSizeAllReturnsEverything covers the value that is not a size. Meshery
// skips the limit entirely for all rather than falling back to the default of
// 25, so a fake that quietly capped at 25 would hide a client bug on any
// collection larger than a page.
//
// The spelling and the fixture size both matter here. /api/pattern reads only
// the lowercase pagesize, so sending pageSize=all would never reach the branch
// under test, and with 25 or fewer designs "no limit" and "defaulted to 25"
// return the same thing.
func TestPageSizeAllReturnsEverything(t *testing.T) {
	s := mesherytest.New(t)
	total := len(s.Data().Designs)
	if total <= 25 {
		t.Fatalf("fixture has %d designs; the test cannot distinguish no-limit from the default below 26", total)
	}

	out := authedGet(t, s, "/api/pattern", "pagesize=all")
	if n := len(out["patterns"].([]any)); n != total {
		t.Fatalf("pagesize=all returned %d designs, want all %d", n, total)
	}

	// The default really is 25, so the assertion above is not vacuous.
	out = authedGet(t, s, "/api/pattern", "")
	if n := len(out["patterns"].([]any)); n != 10 {
		t.Fatalf("no page size returned %d designs, want this endpoint's default of 10", n)
	}

	// And the envelope reports a real size rather than the no-limit sentinel.
	out = authedGet(t, s, "/api/pattern", "pagesize=all")
	if got := out["pageSize"].(float64); got != float64(total) {
		t.Errorf("pageSize = %v with no limit, want the row count %d", got, total)
	}
}

// TestNegativePageIsClamped matches getPaginationParams, which forces a
// negative page to 0 rather than computing a negative offset.
func TestNegativePageIsClamped(t *testing.T) {
	s := mesherytest.New(t)

	out := authedGet(t, s, "/api/pattern", "page=-3&pagesize=2")
	if n := len(out["patterns"].([]any)); n != 2 {
		t.Fatalf("page=-3 returned %d designs, want the first page of 2", n)
	}
	if out["patterns"].([]any)[0].(map[string]any)["id"] != "d-1001" {
		t.Error("page=-3 should have been clamped to the first page")
	}
}

// TestPageBeyondTheEndIsEmptyNotAPanic checks the far edge of the arithmetic.
func TestPageBeyondTheEndIsEmptyNotAPanic(t *testing.T) {
	s := mesherytest.New(t)

	total := len(s.Data().Designs)
	out := authedGet(t, s, "/api/pattern", "page=99&pagesize=25")
	if n := len(out["patterns"].([]any)); n != 0 {
		t.Fatalf("page 99 returned %d designs, want 0", n)
	}
	if n := out["totalCount"].(float64); n != float64(total) {
		t.Fatalf("totalCount = %v, want the unpaginated total of %d", n, total)
	}
}

// TestAsDesignClearsTheFlatList covers the topology path: setting asDesign moves
// the answer into a design and empties resources, so a client that keeps reading
// resources gets an empty list and no error. Meshery's own reference says the
// resources are omitted; this pins that it means emptied rather than absent.
func TestAsDesignClearsTheFlatList(t *testing.T) {
	s := mesherytest.New(t)

	out := authedGet(t, s, "/api/system/meshsync/resources",
		`asDesign=true&clusterIds=["`+s.Data().ClusterID()+`"]`)

	if n := len(out["resources"].([]any)); n != 0 {
		t.Fatalf("resources = %d, want 0: asDesign clears the flat list", n)
	}
	design, ok := out["design"].(map[string]any)
	if !ok {
		t.Fatalf("no design in the response: %v", out)
	}
	if want := len(s.Data().Resources); len(design["components"].([]any)) != want {
		t.Fatalf("components = %d, want %d", len(design["components"].([]any)), want)
	}
}

// TestEnvelopeKeysMatchTheLiveServer pins the response shapes against what a
// running Meshery actually emits. The design key is present whether or not
// asDesign was asked for, because the response struct has no omitempty on it, so
// a client cannot use its presence to tell the two paths apart.
func TestEnvelopeKeysMatchTheLiveServer(t *testing.T) {
	s := mesherytest.New(t)
	cluster := `clusterIds=["` + s.Data().ClusterID() + `"]`

	keys := func(m map[string]any) []string {
		out := make([]string, 0, len(m))
		for k := range m {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}

	plain := keys(authedGet(t, s, "/api/system/meshsync/resources", cluster))
	design := keys(authedGet(t, s, "/api/system/meshsync/resources", cluster+"&asDesign=true"))
	want := []string{"design", "page", "pageSize", "resources", "totalCount"}

	for _, got := range [][]string{plain, design} {
		if !slices.Equal(got, want) {
			t.Errorf("envelope keys = %v, want %v", got, want)
		}
	}

	// And the summary carries labels alongside kinds and namespaces.
	sum := authedGet(t, s, "/api/system/meshsync/resources/summary", "clusterId="+s.Data().ClusterID())
	if !slices.Equal(keys(sum), []string{"kinds", "labels", "namespaces"}) {
		t.Errorf("summary keys = %v", keys(sum))
	}
}

// TestDesignFileEncodingDiffersByEndpoint is the trap no mock reproduces,
// because you only find it by asking a real server twice. The list endpoint
// serves patternFile as YAML and the by-ID endpoint serves the same field as
// JSON, so a client that reads the design out of the list response and decodes
// it as JSON fails on every design it sees.
func TestDesignFileEncodingDiffersByEndpoint(t *testing.T) {
	s := mesherytest.New(t)

	list := authedGet(t, s, "/api/pattern", "pagesize=1")
	listed := list["patterns"].([]any)[0].(map[string]any)["patternFile"].(string)
	if json.Valid([]byte(listed)) {
		t.Error("the list endpoint should serve YAML, which is not valid JSON")
	}
	if !strings.HasPrefix(listed, "name: bookinfo") {
		t.Errorf("list patternFile does not look like YAML: %q", listed[:40])
	}

	byID := authedGet(t, s, "/api/pattern/d-1001", "")
	single := byID["patternFile"].(string)
	if !json.Valid([]byte(single)) {
		t.Errorf("the by-ID endpoint should serve JSON, got %q", single[:40])
	}

	// Same design, two encodings. A client decoding the list form as JSON gets
	// nothing useful out of it.
	var pf map[string]any
	if err := json.Unmarshal([]byte(listed), &pf); err == nil {
		t.Error("expected the YAML form to fail a JSON decode")
	}
	if err := json.Unmarshal([]byte(single), &pf); err != nil {
		t.Fatalf("the by-ID form should decode as JSON: %v", err)
	}
	if pf["name"] != "bookinfo" {
		t.Errorf("name = %v", pf["name"])
	}
}

// TestDesignFileIsAJSONString covers the shape trap on designs. patternFile is
// a JSON string under a camelCase key on current Meshery. Decoding it as a
// nested object, or looking only for pattern_file, yields an empty design with
// no error at all.
func TestDesignFileIsAJSONString(t *testing.T) {
	s := mesherytest.New(t)

	out := authedGet(t, s, "/api/pattern/d-1001", "")
	raw, ok := out["patternFile"].(string)
	if !ok {
		t.Fatalf("patternFile = %T, want a JSON string", out["patternFile"])
	}
	var pf map[string]any
	if err := json.Unmarshal([]byte(raw), &pf); err != nil {
		t.Fatalf("patternFile did not parse as JSON: %v", err)
	}
	if pf["name"] != "bookinfo" {
		t.Fatalf("design name = %v", pf["name"])
	}
	if _, legacy := out["pattern_file"]; legacy {
		t.Error("current Meshery does not serve pattern_file; a client that only reads it should fail here")
	}
}

// TestOrgScopedEndpointsRequireOrgID covers the last silent-400 family.
func TestOrgScopedEndpointsRequireOrgID(t *testing.T) {
	s := mesherytest.New(t)

	for _, path := range []string{"/api/environments", "/api/workspaces"} {
		req, _ := http.NewRequest(http.MethodGet, s.URL()+path, nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: s.Token})
		req.AddCookie(&http.Cookie{Name: "meshery-provider", Value: s.Provider})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400 without orgId", path, resp.StatusCode)
		}
	}
}

// TestRegistryReadsAreUnauthenticated pins the routes that genuinely need no
// session, so a client is not made to authenticate where Meshery does not.
func TestRegistryReadsAreUnauthenticated(t *testing.T) {
	s := mesherytest.New(t)

	status, body := naiveGet(t, s.URL(), "/api/registry/models", "")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if strings.Contains(string(body), "Sign in") {
		t.Fatal("registry GETs are NoAuth in Meshery and should not redirect")
	}
}

// TestRegistryWritesStillNeedASession covers the half of the registry that is
// not NoAuth. Exempting the whole prefix would let an unauthenticated write past
// AssertAuthenticated, which is exactly the kind of hole a blanket rule leaves.
func TestRegistryWritesStillNeedASession(t *testing.T) {
	s := mesherytest.New(t)

	req, _ := http.NewRequest(http.MethodPost, s.URL()+"/api/registry/register", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	// No provider selected, so the provider gate answers before anything looks
	// at the session, and it does that for writes too.
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Sign in") {
		t.Fatalf("an unauthenticated registry write should have hit the provider gate, got: %s", body)
	}

	// And the assertion notices, rather than treating the prefix as exempt.
	failed, msg := (&recorder{}).run(s.AssertAuthenticated)
	if !failed {
		t.Error("AssertAuthenticated exempted an unauthenticated registry write")
	}
	if !strings.Contains(msg, "/api/registry/register") {
		t.Errorf("expected the write to be named in the failure, got: %s", msg)
	}
}

// recorder captures assertion failures instead of reporting them, so a test can
// check that an assertion fires. Fatalf panics with a sentinel, which stands in
// for the runtime.Goexit a real *testing.T would perform.
type recorder struct {
	failures []string
}

type fatal struct{}

func (r *recorder) Helper() {}

func (r *recorder) Errorf(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

func (r *recorder) Fatalf(format string, args ...any) {
	r.Errorf(format, args...)
	panic(fatal{})
}

// run applies an assertion and reports whether it failed, along with the
// message, so a test can check the message explains the trap rather than only
// reporting a mismatch.
func (r *recorder) run(assert func(mesherytest.T)) (failed bool, msg string) {
	// The result is set here rather than after assert returns, because Fatalf
	// unwinds through this defer and never reaches the return below.
	defer func() {
		if p := recover(); p != nil {
			if _, ok := p.(fatal); !ok {
				panic(p)
			}
		}
		failed, msg = len(r.failures) > 0, strings.Join(r.failures, "; ")
	}()
	assert(r)
	return
}

// TestAssertionsFailOnABrokenClient proves the assertions are not vacuous, by
// driving the fake with a client that gets each thing wrong and checking the
// matching assertion fires. Without this, a passing suite would only mean the
// assertions never say anything.
func TestAssertionsFailOnABrokenClient(t *testing.T) {
	cases := []struct {
		name    string
		drive   func(t *testing.T, s *mesherytest.Server)
		assert  func(s *mesherytest.Server) func(mesherytest.T)
		explain string
	}{
		{
			name: "bearer auth instead of cookies",
			drive: func(t *testing.T, s *mesherytest.Server) {
				naiveGet(t, s.URL(), "/api/system/meshsync/resources", "")
			},
			assert:  func(s *mesherytest.Server) func(mesherytest.T) { return s.AssertAuthenticated },
			explain: "Authorization header",
		},
		{
			name: "no cluster filter",
			drive: func(t *testing.T, s *mesherytest.Server) {
				authedGet(t, s, "/api/system/meshsync/resources", "")
			},
			assert: func(s *mesherytest.Server) func(mesherytest.T) {
				return func(tt mesherytest.T) { s.AssertClusterScoped(tt, resourcesPath) }
			},
			explain: "empty IN clause",
		},
		{
			name: "cluster id sent unquoted",
			drive: func(t *testing.T, s *mesherytest.Server) {
				authedGet(t, s, "/api/system/meshsync/resources", "clusterIds=ksid-9c2e")
			},
			assert: func(s *mesherytest.Server) func(mesherytest.T) {
				return func(tt mesherytest.T) { s.AssertClusterScoped(tt, resourcesPath) }
			},
			explain: "not a JSON array",
		},
		{
			name: "summary given the plural spelling",
			drive: func(t *testing.T, s *mesherytest.Server) {
				authedGet(t, s, resourcesPath+"/summary", `clusterIds=["ksid-9c2e"]`)
			},
			assert: func(s *mesherytest.Server) func(mesherytest.T) {
				return func(tt mesherytest.T) { s.AssertClusterScoped(tt, resourcesPath+"/summary") }
			},
			explain: "repeated singular clusterId",
		},
		{
			name: "one-based paging",
			drive: func(t *testing.T, s *mesherytest.Server) {
				authedGet(t, s, "/api/pattern", "page=1")
			},
			assert: func(s *mesherytest.Server) func(mesherytest.T) {
				return func(tt mesherytest.T) { s.AssertZeroBasedPaging(tt, "/api/pattern") }
			},
			explain: "skips the first",
		},
		{
			name:  "endpoint never called",
			drive: func(t *testing.T, s *mesherytest.Server) {},
			assert: func(s *mesherytest.Server) func(mesherytest.T) {
				return func(tt mesherytest.T) { tt.Helper(); s.AssertCalled(tt, "/api/system/kubernetes/contexts") }
			},
			explain: "no request to",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := mesherytest.New(t)
			tc.drive(t, s)

			failed, msg := (&recorder{}).run(tc.assert(s))
			if !failed {
				t.Fatal("the assertion passed on a client that is wrong, so it would not catch this in review")
			}
			if !strings.Contains(msg, tc.explain) {
				t.Errorf("failure message should explain the trap, wanted %q in:\n%s", tc.explain, msg)
			}
		})
	}
}

// The tests below cover the fake's own robustness rather than Meshery's
// behaviour. A test helper that panics, races or leaks is worse than no helper,
// because it fails in someone else's suite and looks like their bug.

// TestDoubleCloseIsSafe checks the exported Close composes with the t.Cleanup
// that New already registers, so calling it explicitly is not a trap.
func TestDoubleCloseIsSafe(t *testing.T) {
	s := mesherytest.New(t)
	s.Close()
	s.Close()
}

// TestConcurrentUse drives the fake from many goroutines while reading the
// recorded requests, which is the shape a parallel suite produces.
func TestConcurrentUse(t *testing.T) {
	s := mesherytest.New(t)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			authedGet(t, s, "/api/pattern", "")
			_ = s.Requests()
			_ = s.Data().ClusterID()
		}()
	}
	wg.Wait()
	if n := len(s.Requests()); n != 20 {
		t.Fatalf("recorded %d requests, want 20", n)
	}
}

// TestGarbagePaginationDoesNotPanic feeds the pagination arithmetic values a
// confused client might send. None of them should reach a slice bounds panic.
func TestGarbagePaginationDoesNotPanic(t *testing.T) {
	s := mesherytest.New(t)
	for _, q := range []string{
		"page=abc&pagesize=xyz",
		"page=999999999999999999999",
		"pagesize=-5",
		"page=&pagesize=",
		"pagesize=0",
		"page=2147483647&pagesize=2147483647",
		// These overflow page*pageSize on a 64-bit int; the third wraps the
		// product negative.
		"page=1&pagesize=9223372036854775807",
		"page=2&pagesize=4611686018427387904",
		"page=4611686018427387904&pagesize=2",
	} {
		req, _ := http.NewRequest(http.MethodGet, s.URL()+"/api/pattern?"+q, nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: s.Token})
		req.AddCookie(&http.Cookie{Name: "meshery-provider", Value: s.Provider})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			// A panic in the handler closes the connection, so a transport
			// error here means the fake crashed rather than answered.
			t.Fatalf("%s: handler did not answer: %v", q, err)
		}
		resp.Body.Close()
		if resp.StatusCode >= 500 {
			t.Errorf("%s: status %d", q, resp.StatusCode)
		}
	}
}

// TestRequestsAreIndependentSnapshots checks a caller cannot reach back into the
// recorded requests through the maps a returned Request carries. A test helper
// that lets one assertion corrupt the next is worse than no helper.
func TestRequestsAreIndependentSnapshots(t *testing.T) {
	s := mesherytest.New(t)
	authedGet(t, s, "/api/pattern", "search=bookinfo")

	first := s.Requests()
	first[0].Query.Set("search", "tampered")
	first[0].Query["kind"] = []string{"injected"}
	first[0].Cookies["token"] = "tampered"
	first[0].Headers["Accept"] = "tampered"

	again := s.Requests()
	if got := again[0].Query.Get("search"); got != "bookinfo" {
		t.Errorf("query survived tampering as %q, want %q", got, "bookinfo")
	}
	if _, ok := again[0].Query["kind"]; ok {
		t.Error("a key added to a returned request reached the recorded one")
	}
	if got := again[0].Cookies["token"]; got != s.Token {
		t.Errorf("cookie = %q, want %q", got, s.Token)
	}
	if got := again[0].Headers["Accept"]; got == "tampered" {
		t.Error("a header written on a returned request reached the recorded one")
	}
}

// TestClusterScopeRejectsAWiderRequest checks the assertion fails when the
// client scoped to more clusters than the test named. Only checking that the
// wanted ids are present would let a client read another cluster's resources
// and still pass.
func TestClusterScopeRejectsAWiderRequest(t *testing.T) {
	s := mesherytest.New(t)
	cluster := s.Data().ClusterID()

	authedGet(t, s, "/api/system/meshsync/resources",
		`clusterIds=["`+cluster+`","someone-elses-cluster"]`)

	failed, msg := (&recorder{}).run(func(tt mesherytest.T) {
		s.AssertClusterScoped(tt, "/api/system/meshsync/resources", cluster)
	})
	if !failed {
		t.Fatal("a request scoped to an extra cluster should not satisfy the assertion")
	}
	if !strings.Contains(msg, "someone-elses-cluster") {
		t.Errorf("the failure should name the extra cluster, got: %s", msg)
	}

	// And the exact scope still passes.
	s2 := mesherytest.New(t)
	authedGet(t, s2, "/api/system/meshsync/resources", `clusterIds=["`+s2.Data().ClusterID()+`"]`)
	if failed, msg := (&recorder{}).run(func(tt mesherytest.T) {
		s2.AssertClusterScoped(tt, "/api/system/meshsync/resources", s2.Data().ClusterID())
	}); failed {
		t.Errorf("the exact scope should pass, got: %s", msg)
	}
}

// TestPublicExemptionStopsAtAPathBoundary checks the registry exemption covers
// the registry and nothing that merely starts with the same letters. A prefix
// match would also cover a sibling like /api/registry-admin, and silently
// exempting a route from the auth check is worse than not checking at all.
func TestPublicExemptionStopsAtAPathBoundary(t *testing.T) {
	s := mesherytest.New(t)

	for _, path := range []string{"/api/registry-admin", "/api/meshmodels-old"} {
		naiveGet(t, s.URL(), path, "")
	}

	failed, msg := (&recorder{}).run(s.AssertAuthenticated)
	if !failed {
		t.Fatal("a sibling path was exempted from the auth check by a prefix match")
	}
	for _, path := range []string{"/api/registry-admin", "/api/meshmodels-old"} {
		if !strings.Contains(msg, path) {
			t.Errorf("expected %s to be flagged, got: %s", path, msg)
		}
	}

	// The real registry subtree and its base are still exempt.
	s2 := mesherytest.New(t)
	naiveGet(t, s2.URL(), "/api/registry/models", "")
	naiveGet(t, s2.URL(), "/api/meshmodels/models", "")
	authedGet(t, s2, "/api/pattern", "")
	if failed, msg := (&recorder{}).run(s2.AssertAuthenticated); failed {
		t.Errorf("registry reads should stay exempt, got: %s", msg)
	}
}

// An empty or null clusterIds array is the empty IN clause this assertion
// exists to catch, so it must fail even when the test names no expected id.
func TestClusterScopedRejectsAnEmptyFilter(t *testing.T) {
	for _, q := range []string{`clusterIds=[]`, `clusterIds=null`, `clusterIds=[""]`} {
		s := mesherytest.New(t)
		authedGet(t, s, "/api/system/meshsync/resources", q)

		failed, msg := (&recorder{}).run(func(tt mesherytest.T) {
			s.AssertClusterScoped(tt, "/api/system/meshsync/resources")
		})
		if !failed {
			t.Errorf("%s: passed, but it filters to nothing", q)
		} else if !strings.Contains(msg, "cluster") {
			t.Errorf("%s: failure should mention the cluster filter, got %s", q, msg)
		}
	}
}

// The summary branch must reject a blank value too, not just an absent key.
func TestClusterScopedRejectsABlankSummaryFilter(t *testing.T) {
	s := mesherytest.New(t)
	authedGet(t, s, "/api/system/meshsync/resources/summary", "clusterId=")

	if failed, _ := (&recorder{}).run(func(tt mesherytest.T) {
		s.AssertClusterScoped(tt, "/api/system/meshsync/resources/summary")
	}); !failed {
		t.Error("a blank clusterId passed the presence check")
	}
}

// A repeated key widens a filter, and AssertQuery must not read only the first.
func TestAssertQueryNoticesARepeatedKey(t *testing.T) {
	s := mesherytest.New(t)
	authedGet(t, s, "/api/integrations/connections", "kind=kubernetes&kind=aws")

	if failed, _ := (&recorder{}).run(func(tt mesherytest.T) {
		s.AssertQuery(tt, "/api/integrations/connections", "kind", "kubernetes")
	}); !failed {
		t.Error("a second kind= value slipped past the assertion")
	}
}

// The paging assertion must judge the value, not its spelling: "00" and "+0"
// are page zero, and a correct client should not be failed for writing one.
//
// The plus is sent percent-encoded on purpose. Written bare in a query string it
// means a space, so the server would receive " 0", which Atoi rejects outright:
// the case would pass through the unparsable branch below without the "+0"
// spelling ever reaching the assertion.
func TestZeroBasedPagingJudgesTheValueNotTheSpelling(t *testing.T) {
	for _, page := range []string{"0", "00", "%2B0"} {
		s := mesherytest.New(t)
		authedGet(t, s, "/api/pattern", "page="+page)
		if failed, msg := (&recorder{}).run(func(tt mesherytest.T) {
			s.AssertZeroBasedPaging(tt, "/api/pattern")
		}); failed {
			t.Errorf("page=%q is page zero but was flagged: %s", page, msg)
		}
	}
	s := mesherytest.New(t)
	authedGet(t, s, "/api/pattern", "page=01")
	if failed, _ := (&recorder{}).run(func(tt mesherytest.T) {
		s.AssertZeroBasedPaging(tt, "/api/pattern")
	}); !failed {
		t.Error("page=01 is page one and should be flagged")
	}

	// A value the server cannot parse is page zero to it, because both
	// pagination paths discard the Atoi error. A bare plus is the everyday way
	// to produce one: in a query string it means a space.
	for _, page := range []string{"+0", "abc", ""} {
		s := mesherytest.New(t)
		authedGet(t, s, "/api/pattern", "page="+page)
		if failed, msg := (&recorder{}).run(func(tt mesherytest.T) {
			s.AssertZeroBasedPaging(tt, "/api/pattern")
		}); failed {
			t.Errorf("page=%q is page zero to the server and should not be flagged: %s", page, msg)
		}
	}
}
