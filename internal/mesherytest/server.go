package mesherytest

import (
	"encoding/base64"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"testing"
)

// DefaultToken and DefaultProvider are the credential values the fake expects,
// matching the two cookies mesheryctl writes to ~/.meshery/auth.json.
const (
	DefaultToken    = "fake-session-jwt"
	DefaultProvider = "Meshery"
)

// Request is one call the fake received, recorded so a test can assert on what
// the client actually sent rather than only on what came back.
type Request struct {
	Method  string
	Path    string
	Query   url.Values
	Cookies map[string]string
	// Headers are keyed with net/http's canonical casing, so look one up with
	// http.CanonicalHeaderKey rather than the wire spelling.
	Headers map[string]string
}

// Server is a fake Meshery Server. Create one with New.
type Server struct {
	// Token is the value of the token cookie the fake accepts. Provider is the
	// provider name, which a request may convey by cookie, header or query.
	Token    string
	Provider string

	// ProviderType selects the authentication behaviour. A local provider
	// accepts everything, mirroring DefaultLocalProvider.GetSession
	// (server/models/default_local_provider.go:482), which takes the request as
	// _ and returns nil. A remote provider requires the cookies.
	ProviderType ProviderType

	// RemoteLoginURL is where an unauthenticated GET is redirected. Empty means
	// the fake's own stand-in at RemoteLoginPath.
	RemoteLoginURL string

	data *Data

	mu       sync.Mutex
	requests []Request

	httpSrv *httptest.Server
}

// ProviderType selects which of Meshery's two authentication behaviours the
// fake reproduces.
type ProviderType int

const (
	// RemoteProvider requires the token and meshery-provider cookies. This is
	// the stricter and more useful default for tests.
	RemoteProvider ProviderType = iota
	// LocalProvider accepts any request. DefaultLocalProvider.GetSession
	// discards the request and returns nil, which is why an incorrect auth
	// implementation can pass against a locally started Meshery and fail
	// against a remote one.
	LocalProvider
)

// Option configures a Server.
type Option func(*Server)

// WithLocalProvider makes the fake accept unauthenticated requests, mirroring
// a locally started Meshery.
func WithLocalProvider() Option {
	return func(s *Server) { s.ProviderType = LocalProvider }
}

// WithRemoteLoginURL overrides where an unauthenticated GET is redirected. Real
// Meshery sends it to the remote provider's own host; the fake serves a local
// stand-in unless this says otherwise.
func WithRemoteLoginURL(url string) Option {
	return func(s *Server) { s.RemoteLoginURL = url }
}

// WithData replaces the seeded fixtures.
func WithData(d *Data) Option {
	return func(s *Server) { s.data = d }
}

// New starts a fake Meshery Server seeded with a small, realistic dataset. The
// server is closed automatically when the test finishes.
func New(t *testing.T, opts ...Option) *Server {
	t.Helper()

	s := &Server{
		Token:    DefaultToken,
		Provider: DefaultProvider,
		data:     SeedData(),
	}
	for _, o := range opts {
		o(s)
	}

	mux := http.NewServeMux()
	s.routes(mux)
	s.httpSrv = httptest.NewServer(s.record(mux))
	t.Cleanup(s.httpSrv.Close)
	return s
}

func (s *Server) remoteLoginURL() string {
	if s.RemoteLoginURL != "" {
		return s.RemoteLoginURL
	}
	return RemoteLoginPath
}

// Data returns the fixtures the fake is serving, so a test can name the seeded
// cluster or design rather than hardcoding an identifier.
func (s *Server) Data() *Data { return s.data }

// URL is the base URL of the fake, suitable as MESHERY_URL.
func (s *Server) URL() string { return s.httpSrv.URL }

// Close shuts the fake down. Tests created via New do not need to call this.
func (s *Server) Close() { s.httpSrv.Close() }

// Requests returns every call the fake received, in order.
func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Request, len(s.requests))
	for i, r := range s.requests {
		// The struct copy alone would leave Query, Cookies and Headers pointing
		// at the recorded maps, so a caller poking at a returned request would
		// change what later assertions read.
		out[i] = r
		out[i].Query = cloneValues(r.Query)
		out[i].Cookies = maps.Clone(r.Cookies)
		out[i].Headers = maps.Clone(r.Headers)
	}
	return out
}

// cloneValues deep-copies query parameters. url.Values holds slices, so a map
// clone alone would still share them.
func cloneValues(v url.Values) url.Values {
	out := make(url.Values, len(v))
	for k, vals := range v {
		out[k] = slices.Clone(vals)
	}
	return out
}

func (s *Server) record(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookies := map[string]string{}
		for _, c := range r.Cookies() {
			cookies[c.Name] = c.Value
		}
		headers := map[string]string{}
		for name := range r.Header {
			headers[name] = r.Header.Get(name)
		}
		s.mu.Lock()
		s.requests = append(s.requests, Request{
			Method:  r.Method,
			Path:    r.URL.Path,
			Query:   r.URL.Query(),
			Cookies: cookies,
			Headers: headers,
		})
		s.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}

// authenticated reports whether the request carries the cookie pair Meshery
// requires. Registry routes bypass this; see routes.
//
// Meshery: RemoteProvider.GetToken reads req.Cookie(TokenCookieName) and
// returns an error when it is absent, and mesheryctl sends token alongside
// meshery-provider. Meshery does set Authorization headers, but on its own
// outbound calls; GetToken is what an inbound request is judged by.
func (s *Server) authenticated(r *http.Request) bool {
	if s.ProviderType == LocalProvider {
		return true
	}
	tok, err := r.Cookie("token")
	if err != nil || tok.Value != s.Token {
		return false
	}
	return resolveProvider(r) == s.Provider
}

// resolveProvider mirrors Meshery's provider selection precedence.
//
// Meshery: resolveProviderName (server/handlers/middlewares.go:56-73) takes the
// meshery-provider cookie, else an HTTP header of the same name, else the
// ?provider= query parameter. Note the asymmetry with the token, which
// RemoteProvider.GetToken reads from the cookie and nowhere else: the provider
// has three channels, the session has one.
//
// Not reproduced: when the PROVIDER environment variable is set on the server,
// it wins outright and all three client hints are ignored.
func resolveProvider(r *http.Request) string {
	if ck, err := r.Cookie("meshery-provider"); err == nil && ck.Value != "" {
		return ck.Value
	}
	if hdr := r.Header.Get("meshery-provider"); hdr != "" {
		return hdr
	}
	return r.URL.Query().Get("provider")
}

// RemoteLoginPath is where the fake sends an unauthenticated GET, standing in
// for the remote provider's own login page.
//
// Real Meshery redirects off-host here, to RemoteProviderURL + "/login", which
// is the provider's server and not Meshery's. The fake serves the stand-in
// itself so tests stay hermetic and do not depend on name resolution. Use
// WithRemoteLoginURL to point it somewhere else if the off-host hop is the thing
// under test.
const RemoteLoginPath = "/remote-provider/login"

// rejectUnauthenticated reproduces what Meshery actually does with a request
// that fails authentication, which is not one behaviour but three, and which
// one you get depends on whether a token cookie was sent at all.
//
// No token cookie is the common case and it does not take the path most people
// assume. RemoteProvider.GetSession returns ErrEmptySession when GetToken finds
// no cookie, and AuthMiddleware explicitly excludes ErrEmptySession from the
// HandleUnAuthenticated branch:
//
//	if !errors.Is(err, models.ErrEmptySession) && provider.GetProviderType() == models.RemoteProviderType {
//		provider.HandleUnAuthenticated(w, req)
//
// so it falls through to LoginHandler instead. LoginHandler answers a non-GET
// with a bare 404 and sends a GET to InitiateLogin, which redirects to the
// remote provider's own login URL. Two consequences worth a test: an
// unauthenticated GET is redirected off-host, and an unauthenticated POST looks
// like a missing endpoint rather than a missing session.
//
// A token cookie that is present but not valid is the case that does reach
// HandleUnAuthenticated, which redirects to /auth/login when the
// meshery-provider cookie is present and to /provider when it is not.
//
// Deliberately not reproduced: HandleUnAuthenticated counts attempts in a cookie
// and answers 401 once retries reach MaxAuthRetries, which is 3, and
// AuthMiddleware answers 401 outright on an enforced-provider mismatch. Both are
// states a client reaches only after the ones above.
func (s *Server) rejectUnauthenticated(w http.ResponseWriter, r *http.Request) {
	// The provider gate comes first and it does not care about the method.
	// AuthMiddleware pulls the provider out of the request context and, when
	// nothing valid is there, redirects to /provider carrying the original path
	// base64-encoded in ref. Verified live: an unauthenticated GET, POST and
	// DELETE all came back 302 to /provider?ref=..., before any session check.
	if resolveProvider(r) != s.Provider {
		ref := base64.RawURLEncoding.EncodeToString([]byte(r.URL.Path))
		http.Redirect(w, r, "/provider?ref="+ref, http.StatusFound)
		return
	}

	// Past the gate, a missing token is ErrEmptySession, which AuthMiddleware
	// excludes from HandleUnAuthenticated, so it lands in LoginHandler: a
	// non-GET gets a bare 404 and a GET goes to the provider's login page.
	// Read from the source rather than observed, because reaching it needs a
	// reachable remote provider.
	if _, err := r.Cookie("token"); err != nil {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.Redirect(w, r, s.remoteLoginURL(), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/auth/login", http.StatusFound)
}

// Provider returns how the request conveyed its provider selection, and by
// which of Meshery's three channels.
func (r Request) Provider() (value, channel string) {
	if v := r.Cookies["meshery-provider"]; v != "" {
		return v, "cookie"
	}
	if v := r.Headers[http.CanonicalHeaderKey("meshery-provider")]; v != "" {
		return v, "header"
	}
	if v := r.Query.Get("provider"); v != "" {
		return v, "query"
	}
	return "", ""
}

// loginPage is what a redirect-following client actually receives: HTML with a
// 200, which is why the failure surfaces as a JSON parse error.
func loginPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("<!doctype html><html><body>Sign in to Meshery</body></html>"))
}

// unlimited is the page size that means "no limit", which is what Meshery does
// for pageSize=all.
const unlimited = -1

// paginate applies Meshery's pagination arithmetic.
//
// Meshery: getPaginationParams computes offset = page * limit
// (server/handlers/utils.go:116), and models.Paginate does
// offset := (page) * pageSize (server/models/persister_utils.go:10). Both are
// zero-based, so page=1 skips the first page. Meshery's own callers open with
// page := 0. Negative pages are clamped to 0, as getPaginationParams does.
func paginate(total, page, pageSize int) (start, end int) {
	if page < 0 {
		page = 0
	}
	if pageSize == unlimited {
		return 0, total
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	// Compare before multiplying. page * pageSize overflows int for large
	// values a confused client can send, and an overflowed product can land
	// negative, which would be a slice bounds panic inside someone else's test
	// run rather than an empty page.
	if page > total/pageSize {
		return total, total
	}
	start = page * pageSize
	end = start + pageSize
	if end > total || end < 0 {
		end = total
	}
	return start, end
}

const defaultPageSize = 25

// pageStyle is how one endpoint handles paging. Meshery is not consistent about
// either half, and neither is announced: an endpoint that reads only the
// lowercase spelling ignores pageSize and quietly applies its own default, and
// that default is not the same everywhere.
type pageStyle struct {
	// readsCamelCase is true where the handler reads pageSize and falls back to
	// pagesize, false where it reads pagesize and nothing else.
	readsCamelCase bool
	// defaultSize is what the endpoint applies when no page size is given.
	defaultSize int
}

// pageStyles records what each endpoint the fake serves actually does. Measured
// against a Meshery Server built from master and run locally, by asking each
// endpoint with pagesize, with pageSize, and with neither.
//
//	                                  camelCase   default
//	/api/pattern                         no          10
//	/api/system/kubernetes/contexts      no          10
//	/api/identity/orgs                   no          10
//	/api/integrations/connections        yes         10
//	/api/system/meshsync/resources       yes         25
//	/api/registry/models                 yes         25
//
// The two halves are independent: connections reads the camelCase spelling and
// still defaults to 10, so a client cannot infer one from the other.
var pageStyles = map[string]pageStyle{
	"/api/system/kubernetes/contexts": {readsCamelCase: false, defaultSize: 10},
	"/api/pattern":                    {readsCamelCase: false, defaultSize: 10},
	"/api/identity/orgs":              {readsCamelCase: false, defaultSize: 10},
	"/api/integrations/connections":   {readsCamelCase: true, defaultSize: 10},
	"/api/system/meshsync/resources":  {readsCamelCase: true, defaultSize: 25},
}

// styleFor returns the paging behaviour for a path. Anything unlisted falls back
// to getPaginationParams, which reads both spellings and defaults to 25.
func styleFor(path string) pageStyle {
	if st, ok := pageStyles[path]; ok {
		return st
	}
	return pageStyle{readsCamelCase: true, defaultSize: defaultPageSize}
}

// pageParams reads the pagination parameters the way the given endpoint does.
//
// A client that sends pageSize to an endpoint reading only pagesize gets the
// default, which varies by endpoint, and no error.
// reportedSize is the page size to echo in a response envelope. There is no
// limit to report when the caller asked for everything, so the row count stands
// in rather than leaking the sentinel.
func reportedSize(pageSize, returned int) int {
	if pageSize == unlimited {
		return returned
	}
	return pageSize
}

func pageParams(q url.Values, style pageStyle) (page, pageSize int) {
	page, _ = strconv.Atoi(q.Get("page"))
	sizeStr := q.Get("pagesize")
	if style.readsCamelCase {
		if v := q.Get("pageSize"); v != "" {
			sizeStr = v
		}
	}
	// Meshery's persisters special-case pageSize=all to fetch every row rather
	// than applying a limit, so it is not a page size at all.
	if sizeStr == "all" {
		return page, unlimited
	}
	pageSize, _ = strconv.Atoi(sizeStr)
	if pageSize <= 0 {
		pageSize = style.defaultSize
	}
	return page, pageSize
}

// clusterFilter is the outcome of reading the clusterIds query parameter. The
// three cases are genuinely different and Meshery treats them differently.
type clusterFilter int

const (
	// clusterFilterAbsent: no clusterIds at all. Meshery sets the filter to an
	// empty slice and binds it into cluster_id IN (?), which matches nothing
	// and still answers 200. This is the silent one.
	clusterFilterAbsent clusterFilter = iota
	// clusterFilterMalformed: present but not a JSON array. Meshery answers 400.
	clusterFilterMalformed
	// clusterFilterPresent: a well-formed JSON array.
	clusterFilterPresent
)

// parseClusterIDs reads the JSON-encoded array that
// /api/system/meshsync/resources expects.
//
// Meshery: server/handlers/meshsync_handler.go:267-278 does
// Query().Get("clusterIds"), json.Unmarshals it into a []string, answers 400 if
// that fails, and otherwise sets filter.ClusterIds = []string{} when the
// parameter is absent. Line 283 then builds
// Where("kubernetes_resources.cluster_id IN (?)", filter.ClusterIds).
func parseClusterIDs(q url.Values) ([]string, clusterFilter) {
	raw := q.Get("clusterIds")
	if raw == "" {
		return nil, clusterFilterAbsent
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, clusterFilterMalformed
	}
	return ids, clusterFilterPresent
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
