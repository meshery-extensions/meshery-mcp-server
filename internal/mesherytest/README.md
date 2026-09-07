# mesherytest

A fake Meshery Server for testing Meshery clients, including MCP servers.

It is a real `httptest.Server` that reproduces Meshery's actual behaviour on the
endpoints an MCP server reads, quirks included, and then lets a test assert on
what the client sent rather than only on what came back.

```go
func TestMyTool(t *testing.T) {
	fake := mesherytest.New(t)
	client := myclient.New(fake.URL(), fake.Token, fake.Provider)

	out, err := client.ListResources(ctx, fake.Data().ClusterID())
	// ... assert on out ...

	fake.AssertAuthenticated(t)
	fake.AssertClusterScoped(t, "/api/system/meshsync/resources", fake.Data().ClusterID())
	fake.AssertZeroBasedPaging(t, "/api/pattern")
}
```

## Why

Meshery has several endpoints that answer a wrong request with `200 OK` and
nothing in it. `GET /api/system/meshsync/resources` filters with
`cluster_id IN (?)`, so a missing filter binds an empty slice and the cluster
reads as empty. Pagination is zero-based on both of Meshery's offset paths, so a
client that opens at page 1 skips the first page of every list. Half the
measured endpoints read only the lowercase `pagesize` and ignore `pageSize`
without saying so, and the defaults differ too. None
of these produce an error, and an AI client presents every one of them as an
answer.

A hand-written mock cannot catch any of them, because it returns the shape the
code under test expects. The mock agrees with the code, including where the code
is wrong about Meshery. The test passes, the reviewer sees green, and the failure
shows up the first time someone points the thing at a real server.

## What it reproduces

The behaviours below come from `meshery/meshery` at commit `e6ed2de`. Most are
reproduced by this fake and cited in `doc.go`; a few are recorded as context.

**Authentication.** The two credentials are not symmetric.
`RemoteProvider.GetToken` reads `req.Cookie("token")` and nothing else, so an
`Authorization` header is not a session and neither is a token on the query
string. The provider selection has three channels: the `meshery-provider`
cookie, else an HTTP header of the same name, else `?provider=`. The fake
accepts all three for the provider and only the cookie for the token, and
`AssertAuthenticated` reports which channel carried it.

**Failing authentication is not one behaviour, and the first thing you meet is
not the session check.** With no provider selected, `AuthMiddleware` redirects to
`/provider` with the original path base64-encoded in `ref`, and it does that for
every method. Verified live: an unauthenticated GET, POST and DELETE all came
back `302` to `/provider?ref=...`. A client that has not chosen a provider never
reaches the session logic at all.

Past that gate it depends on whether a token cookie was sent. With none,
`RemoteProvider.GetSession` returns `ErrEmptySession`, which `AuthMiddleware`
explicitly excludes from the `HandleUnAuthenticated` branch, so the request falls
through to `LoginHandler`: a non-GET gets a bare `404`, and a GET is redirected
off-host to the remote provider's own login URL. A token that is present but
invalid is the case that does reach `HandleUnAuthenticated`, which stays on
Meshery and redirects to `/auth/login` (or to `/provider` when the provider
arrived by header or query rather than cookie, a branch the fake does not
model).

The two outcomes fail differently. Anything that redirects, which is the gate,
the no-token GET and the invalid-token case, ends with a redirect-following
client holding `200 OK` and HTML, failing inside its JSON decoder. The no-token
non-GET does not redirect at all: it is a bare `404` that reads like a missing
endpoint rather than a missing session.

The fake serves its own stand-in for the provider login page so tests stay
hermetic; `WithRemoteLoginURL` points it off-host if that hop is what you are
testing. `WithLocalProvider()` switches to the local provider's behaviour, which
accepts everything, because that is why a client with broken auth can pass its
whole suite against a locally started Meshery and fail against a remote one.

Two 401 paths are deliberately not reproduced, both documented in `doc.go`:
`HandleUnAuthenticated` counts attempts in a cookie and answers 401 once retries
reach `MaxAuthRetries` (3), and `AuthMiddleware` answers 401 outright on an
enforced-provider mismatch. Both are states a client reaches only after the ones
above.

**Cluster scoping.** `/resources` takes a JSON-encoded `clusterIds` array. Omit
it and the handler binds an empty slice into `cluster_id IN (?)`, which matches
no rows, so you get `200 OK` with nothing. Send a bare unquoted id instead and
`json.Unmarshal` fails, which is a 400. The silent case and the loud
case are different, and the fake keeps them different. Its sibling
`/resources/summary` takes a repeated singular `clusterId` and answers 400
without it, so the two spellings are not interchangeable either.

**Pagination.** Zero-based on both of Meshery's offset computations
(`offset = page * limit` and `offset := (page) * pageSize`). Negative pages are
clamped to 0, and `pageSize=all` skips the limit entirely rather than falling
back to the default.

**Both halves of paging are per endpoint, and neither is announced.** Measured
against a running server by asking each endpoint with `pagesize`, with
`pageSize`, and with neither:

| endpoint | reads `pageSize` | default |
|---|---|---|
| `/api/pattern` | no | 10 |
| `/api/system/kubernetes/contexts` | no | 10 |
| `/api/identity/orgs` | no | 10 |
| `/api/integrations/connections` | yes | 10 |
| `/api/system/meshsync/resources` | yes | 25 |
| `/api/registry/models` | yes | 25 |

The two columns are independent. Connections reads the camelCase spelling and
still defaults to 10, so a client cannot infer one from the other, and one that
assumes 25 everywhere mis-pages against four of these six.

**On the spelling specifically.** Of
the endpoints this package serves, the contexts and designs handlers read
`q.Get("pagesize")` and ignore `pageSize`, while `getPaginationParams` and
`GetConnections` read `pageSize` and fall back to `pagesize`. A client sending
`pageSize` to the first group has it ignored and gets that endpoint's own
default instead, which is 10 for all of them, as the table above sets out. Elsewhere in Meshery there are at least two further spellings,
`FetchSmiResult` reading only `pageSize` and the credentials handler reading
`page_size`, so this is a per-endpoint fact rather than a rule.

The fake reproduces the spelling each endpoint it serves actually reads rather
than accepting both everywhere, because accepting both is precisely how a mock
hides this. `AssertPageSizeSpelling` names the spelling that endpoint reads. The
lowercase spelling is safe on every endpoint the fake serves.

**Designs.** `patternFile` is a *string* under a camelCase key, and which
encoding depends on the endpoint: YAML from the list, JSON by ID, as the
live-verified section below sets out. Decoding it as a nested object, reading
only the older `pattern_file`, or handling only one encoding, yields an empty
design and no error.

**Topology.** `?asDesign=true` empties the flat `resources` list and returns a
component graph instead, so the two are never both populated. Evaluation runs at
depth 1 and falls back to the un-evaluated design on failure while still
answering 200, which means an empty `relationships` array does not distinguish
"no edges" from "evaluation failed".

**Org scoping.** `/api/environments` answers 400 without an `orgId`, and
`/api/workspaces` answers 400 only when both `orgId` and the legacy `orgID` are
absent.

**Registry auth is a mix, not a blanket.** Every `GET` is registered with
`models.NoAuth` and needs no session, which is why a read-only client should not
be made to authenticate there. Several mutating routes are registered with
`models.ProviderAuth` and do need one, among them `POST /register`,
`DELETE /models/{id}`, `POST /relationships/evaluate` and the
connection-definition writes. `AssertAuthenticated` exempts the reads and not the
writes, because exempting the whole prefix would let an unauthenticated write
through.

Both prefixes count. `registerRegistryRoute` binds every registry subpath at
`/api/registry` and again at the deprecated `/api/meshmodels` alias, same handler
and same methods, so the fake serves both and the assertion exempts both. Demanding
cookies on the alias would fail a client that is doing nothing wrong, which is the
worst thing an assertion can do.

## Verified against a running Meshery

The behaviours here started from `meshery/meshery` source at commit `e6ed2de`,
and the ones observable from outside were then checked against a Meshery Server
built from that same commit and run locally.

Building it is the part that has stopped people. The published image is amd64
only and dies under emulation on arm64 during content seeding. The source builds
and runs natively, and Meshery's `go.mod` targets Go 1.26.4:

```bash
cd meshery/server/cmd && go build -o meshery-server .
PORT=9081 PROVIDER=Local KEYS_PATH=../../server/permissions/keys.csv ./meshery-server
```

It seeds itself with roughly 355 designs and 292 models, which is enough to exercise
pagination and the design endpoints for real. Confirmed:

```text
GET /api/pattern                        (no provider)    -> 302 /provider?ref=L2FwaS9wYXR0ZXJu
POST /api/pattern                       (no provider)    -> 302 /provider?ref=L2FwaS9wYXR0ZXJu
GET /api/pattern    (meshery-provider=Local, or None)    -> 200
GET /api/system/meshsync/resources                       -> 200 {"page":0,"pageSize":25,"totalCount":0,"resources":[],"design":{...}}
GET /api/system/meshsync/resources?clusterIds=not-json   -> 400
GET /api/system/meshsync/resources/summary               -> 400
GET /api/system/meshsync/resources/summary?clusterId=abc -> 200
GET /api/environments                                    -> 400
GET /api/workspaces                                      -> 400
GET /api/pattern?page=0&pagesize=3                       -> first three designs
GET /api/pattern?page=1&pagesize=3                       -> a different three
```

So the envelope is camelCase on the wire, `pageSize` and `totalCount`, and the
pager really is zero-based, with `page=1` returning the second page. Both
`resources` and `design` are always present rather than one or the other: the key
set is identical with and without `asDesign`, since the response struct carries
no `omitempty` on `design`, so a client cannot use its presence to tell the two
paths apart. The summary carries `labels` alongside `kinds` and `namespaces`, and
all three come back `null` rather than `[]` on a cluster holding nothing.

Registry routing checked as well. `/api/registry/{models,categories,relationships,registrants}`
and the same four paths under `/api/meshmodels` all answer 200, a registry GET
with no cookies at all answers 200, and `POST /api/registry/register` with no
cookies is redirected. That is exactly the split `AssertAuthenticated` encodes.

**The design file is YAML from the list endpoint and JSON from the by-ID
endpoint.** Six designs checked, all six the same way:

```text
design                             list   by-id
prometheus-postgres-exporter       YAML   JSON
Pod Resource Memory Request Limit  YAML   JSON
Pod Liveness                       YAML   JSON
Kubernetes Deployment with Azure   YAML   JSON
Edge Reference Relationship        YAML   JSON
ZooKeeper Cluster                  YAML   JSON
```

`SaveMesheryPattern` stores the design with `yaml.Marshal`
(`server/models/meshery_pattern_persister.go:233`) and the list path returns that
stored form verbatim. A client that reads the design out of a list response and
decodes it as JSON fails on every design, and passes against any mock that
serves JSON from both. This is the one behaviour here that reading the source
did not reveal, because you only find it by asking a real server for the same
design twice. The fake serves YAML from the list and JSON by ID, so a test can
catch it.

`Local` is the canonical local provider name and `None` is still accepted as the
legacy alias; both selected it. The local provider then answered an
unauthenticated request with `200`, which is the behaviour `WithLocalProvider()`
reproduces and the reason broken auth can pass a whole suite locally.

Not covered by the live run: the remote-provider session paths, which need a
reachable remote provider, so the `/auth/login` redirect and the bare `404` on a
non-GET past the gate are still read from the source rather than observed. Nor
anything needing a real Kubernetes cluster. MeshSync
wants an operator in-cluster, so the cluster-scoped endpoints were exercised
against their guards and their empty shapes rather than against discovered
workloads.

## Assertions

| Call | Catches |
|---|---|
| `AssertAuthenticated` | header auth, a token off the cookie, no provider on any channel |
| `AssertClusterScoped` | absent filter, bare id where an array is required, wrong spelling per endpoint |
| `AssertZeroBasedPaging` | a client that opens at page 1 |
| `AssertPageSizeSpelling` | `pageSize` sent where only `pagesize` is read |
| `AssertQuery` / `AssertNoQuery` | a dropped filter; a field that should never be requested |
| `AssertCalled` / `AssertNotCalled` | an endpoint skipped; a mutating route touched by a read-only server |

Assertions take a `mesherytest.T` interface rather than `*testing.T`, which is
how the package's own tests check that each one fires on a client that is wrong.
`*testing.T` satisfies it.

## Status

The package is stdlib-only, so it carries no dependencies into whatever
repository holds it. One test is not here: the positive control that drives a
real Meshery client through every assertion at once. It needs a client to point
at, so it lives alongside one. Apache 2.0, matching Meshery.
