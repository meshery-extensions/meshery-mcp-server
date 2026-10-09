# Testing against a real Meshery Server

Everything in this repository is currently tested against mocks. That is not
anyone's preference, it is a consequence: Meshery publishes an amd64-only image,
and under emulation on arm64 it crashes during content seeding, so a lot of
contributors simply cannot get one running.

Building the server from source works, and takes about two minutes.

```bash
./scripts/meshery-dev-server.sh
```

It fetches `meshery/meshery` at a pinned revision into `.meshery-src`, builds
`server/cmd`, and serves on port 9081 with the built-in `Local` provider. Pass a
path to use a checkout you already have.

The pin is `e6ed2de164b42d805b78dd1cdb3c4b415e8686eb`, the revision every
observation below was made against. Seed counts and endpoint behaviour are
specific to it. Set `MESHERY_REF` to test a newer Meshery, and treat any
difference from what is recorded here as a finding about Meshery rather than
about the tests.

Two things to be accurate about. Setup needs network access, for the clone and
for a large Go module tree; it is only the running server that needs no remote
provider and no credentials, because `PROVIDER=Local` selects the built-in one.

And it listens on every interface rather than loopback. Meshery offers no
bind-address option: `server/router/server.go` passes `fmt.Sprintf(":%d", port)`
straight to `http.ListenAndServe`. So this is a development server, and running
it on an untrusted network exposes it. Firewall the port or stay on a trusted
one.

It seeds itself with roughly 355 designs and 292 models, which is enough to
exercise pagination and the design endpoints against real volume rather than a
two-row fixture.

## Why it is worth the two minutes

A mock can only be as correct as its author's reading of the API, and it returns
the shape the code under test expects. That is comfortable and it hides a
specific class of mistake. Some of what a live server says that a mock will not:

**The design file is served in two different encodings.** `GET /api/pattern`
returns `patternFile` as YAML; `GET /api/pattern/{id}` returns the same field as
JSON. Six designs checked, all six the same way. `SaveMesheryPattern` stores the
design with `yaml.Marshal` and the list path returns that stored form verbatim. A
client that decodes the list form as JSON fails on every design and passes
against any mock, because nobody writes a mock that serves one field two ways.

**Paging is zero-based, and both halves of it vary per endpoint.** Asking each
endpoint with `pagesize`, with `pageSize`, and with neither:

| endpoint | reads `pageSize` | default |
|---|---|---|
| `/api/pattern` | no | 10 |
| `/api/system/kubernetes/contexts` | no | 10 |
| `/api/identity/orgs` | no | 10 |
| `/api/integrations/connections` | yes | 10 |
| `/api/system/meshsync/resources` | yes | 25 |
| `/api/registry/models` | yes | 25 |

The two columns are independent, so one cannot be inferred from the other, and a
client assuming a single default mis-pages against four of these six.

**Wrong requests answer `200` with nothing.** `GET /api/system/meshsync/resources`
without `clusterIds` binds an empty slice into `cluster_id IN (?)`, matches no
rows, and returns success. With a malformed `clusterIds` it is a `400`. The two
failures look nothing alike and only one of them is loud.

**Authentication answers before the session check.** With no provider selected,
an unauthenticated `GET`, `POST` and `DELETE` all come back `302` to
`/provider?ref=<base64 path>`, before any session logic runs. `Local` is the
canonical local provider name and `None` is still accepted as the legacy alias.

## Writing tests against it

Put them behind a build tag so the default suite stays hermetic and needs no
server:

```go
//go:build integration
```

```bash
MESHERY_URL=http://127.0.0.1:9081 go test -tags integration ./...
```

Skip rather than fail when the variable is unset, so `go test ./...` is still the
command a contributor runs without setup.

## What this does not give you

A Kubernetes cluster. MeshSync needs an operator running in-cluster, so without
one the cluster-scoped endpoints can be exercised against their guards and their
empty shapes but not against discovered workloads. That is worth knowing before
writing a test that expects resources back: an empty result there is the correct
answer, not a failure.
