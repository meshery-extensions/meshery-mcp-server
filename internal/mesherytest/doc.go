// Package mesherytest provides a fake Meshery Server for testing Meshery
// clients, including MCP servers.
//
// It exists because Meshery has a set of behaviours that a hand-written mock
// will not reproduce unless the author already knows about them, and most of
// them fail by returning plausible data rather than an error. A mock returns the
// shape the code under test expects, so it agrees with the code even where the
// code is wrong about Meshery: the test passes, and the same code returns
// nothing against a real instance.
//
// Every behaviour below was verified against meshery/meshery at commit
// e6ed2de164b42d805b78dd1cdb3c4b415e8686eb and is cited at the code that
// reproduces it. Line numbers are given against that commit and will drift;
// the function names are the durable part. Where Meshery's behaviour is
// narrower than a tidy rule would suggest, the narrow version is what is
// reproduced, and the difference is written down rather than smoothed over.
//
// Pagination
//
//   - Zero-based on both of Meshery's offset computations: offset = page * limit
//     (server/handlers/utils.go:116) and offset := (page) * pageSize
//     (server/models/persister_utils.go:10). Requesting page=1 skips the first
//     page rather than returning it.
//   - Negative pages are clamped to 0. The default page size is per endpoint,
//     not global: 10 on the provider-backed list endpoints (designs, contexts,
//     connections, organizations) and 25 on the ones behind getPaginationParams
//     (meshsync resources, registry). Measured against a running server.
//   - pageSize=all skips the limit entirely rather than falling back to 25.
//   - The page-size parameter is spelled differently per endpoint, silently.
//     Of the endpoints this package serves, the contexts and designs handlers
//     read q.Get("pagesize") and ignore pageSize, while getPaginationParams and
//     GetConnections read pageSize and fall back to pagesize. Elsewhere in the
//     tree there are at least two more spellings: FetchSmiResult reads only
//     pageSize, and the credentials handler reads page_size. The fake models
//     the endpoints it serves; see pageStyles and styleFor in server.go. The
//     lowercase spelling is safe on every endpoint the fake serves.
//
// Cluster scoping
//
//   - GET /api/system/meshsync/resources reads clusterIds with Query().Get and
//     json.Unmarshals that one value into a []string
//     (server/handlers/meshsync_handler.go:267-278). A bare unquoted id fails to
//     parse and is answered with 400.
//   - When clusterIds is absent the filter becomes an empty slice, bound into
//     Where("kubernetes_resources.cluster_id IN (?)") at line 283, which matches
//     no rows and still answers 200. This is the silent one.
//   - GET /api/system/meshsync/resources/summary reads Query()["clusterId"], a
//     repeated singular parameter, and answers 400 when the key is absent
//     (lines 456-463). The two sibling endpoints disagree on both spelling and
//     encoding, and the summary guard is a presence check: clusterId= and
//     clusterId=all both pass it.
//   - Resource rows are keyed on the value GET /api/system/kubernetes/contexts
//     calls kubernetesServerId, which is neither connectionId nor the context id.
//
// Authentication
//
//   - The token and the provider are not symmetric. RemoteProvider.GetToken
//     (server/models/remote_auth.go:191) reads req.Cookie("token") and nothing
//     else, so an Authorization header is not a session. The provider is
//     resolved from the meshery-provider cookie, else a header of the same name,
//     else ?provider= (server/handlers/middlewares.go:64-71).
//   - No inbound route reads an Authorization header to establish a session.
//   - Failing authentication is not one behaviour but three, and which one you
//     get depends on whether a token cookie was sent. With none,
//     RemoteProvider.GetSession returns ErrEmptySession, which AuthMiddleware
//     explicitly excludes from the HandleUnAuthenticated branch
//     (server/handlers/middlewares.go:174), so the request falls through to
//     LoginHandler: a non-GET gets a bare 404
//     (server/handlers/common_handlers.go:19), and a GET is redirected
//     off-host to the remote provider's own login URL
//     (server/models/remote_provider.go:685). A token that is present but
//     invalid is the case that does reach HandleUnAuthenticated
//     (server/models/remote_provider.go:1049), which redirects to /auth/login
//     when the meshery-provider cookie is present and to /provider when it is
//     not. Either way a redirect-following client gets 200 with HTML and fails
//     in its JSON decoder rather than reporting an auth problem.
//     Deliberately not reproduced: HandleUnAuthenticated counts attempts in a
//     cookie and answers 401 once retries reach MaxAuthRetries, which is 3
//     (server/models/remote_auth.go:39), and AuthMiddleware answers 401 outright
//     on an enforced-provider mismatch (server/handlers/middlewares.go:169).
//   - DefaultLocalProvider.GetSession (server/models/default_local_provider.go:482)
//     takes the request as _ and returns nil, so a locally started Meshery
//     accepts anything. That is why broken auth can pass a whole suite locally
//     and fail against a remote provider. WithLocalProvider selects it.
//
// Designs and topology
//
//   - MesheryPattern.PatternFile is a string under the camelCase key
//     patternFile (server/models/meshery_pattern.go:91), not a nested object,
//     and its encoding varies by endpoint: YAML from GET /api/pattern, JSON
//     from GET /api/pattern/{id}, measured six designs for six against a live
//     server. Older releases spelled the key pattern_file.
//   - asDesign=true clears the flat resources list and returns a component graph
//     instead. Meshery's published v0.9 REST reference puts it as "If true then
//     the response is returned as a design and resources are omitted". Both keys
//     are still emitted; resources is emptied rather than dropped.
//   - Relationship evaluation runs at depth 1, and on failure the handler falls
//     back to the un-evaluated design and still answers 200, so an empty
//     relationships array does not distinguish "no edges" from "evaluation
//     failed".
//
// Scoping and auth of the rest
//
//   - /api/environments answers 400 without orgId. /api/workspaces answers 400
//     only when both orgId and the legacy orgID are absent.
//   - Registry auth is a mix. Every GET is registered with models.NoAuth, but
//     several writes are registered with models.ProviderAuth, among them
//     POST /register, DELETE /models/{id} and POST /relationships/evaluate.
//     registerRegistryRoute (server/router/server.go:24) binds every registry
//     subpath at /api/registry and again at the deprecated /api/meshmodels
//     alias, same handler and same methods, so a client using the alias is not
//     doing anything wrong.
//   - List envelopes on the data endpoints are camelCase (page, pageSize,
//     totalCount). /api/identity/orgs is different: its custom MarshalJSON
//     (server/models/organization.go:31-42) emits both spellings during a
//     deprecation window. The environments and workspaces responses are proxied
//     from the provider rather than built by Meshery, so their envelope is not
//     modelled here beyond the orgId guard those two stubs exist to demonstrate.
//
// Typical use:
//
//	srv := mesherytest.New(t)
//	client := yourclient.New(srv.URL(), srv.Token, srv.Provider)
//
//	if _, err := client.ListDesigns(ctx); err != nil {
//		t.Fatal(err)
//	}
//	srv.AssertAuthenticated(t)
//	srv.AssertZeroBasedPaging(t, "/api/pattern")
//	srv.AssertPageSizeSpelling(t, "/api/pattern")
package mesherytest
