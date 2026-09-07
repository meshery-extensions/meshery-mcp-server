package mesherytest

import (
	"net/http"
	"sort"
	"strings"
)

func (s *Server) routes(mux *http.ServeMux) {
	// The login page a redirect-following client lands on.
	mux.HandleFunc("/provider", loginPage)
	mux.HandleFunc("/auth/login", loginPage)
	mux.HandleFunc(RemoteLoginPath, loginPage)

	// Unauthenticated by design, matching Meshery.
	mux.HandleFunc("/api/system/version", s.handleVersion)
	// registerRegistryRoute binds every registry subpath at both prefixes, with
	// the same handler and the same methods, so the alias is not optional.
	for _, prefix := range []string{"/api/registry", "/api/meshmodels"} {
		mux.HandleFunc(prefix, s.handleRegistry)
		mux.HandleFunc(prefix+"/", s.handleRegistry)
	}

	mux.HandleFunc("/api/system/kubernetes/contexts", s.guard(s.handleContexts))
	mux.HandleFunc("/api/integrations/connections", s.guard(s.handleConnections))
	mux.HandleFunc("/api/system/meshsync/resources", s.guard(s.handleResources))
	mux.HandleFunc("/api/system/meshsync/resources/summary", s.guard(s.handleSummary))
	mux.HandleFunc("/api/pattern", s.guard(s.handlePatterns))
	mux.HandleFunc("/api/pattern/", s.guard(s.handlePatternByID))
	mux.HandleFunc("/api/environments", s.guard(s.handleEnvironments))
	mux.HandleFunc("/api/workspaces", s.guard(s.handleWorkspaces))
	mux.HandleFunc("/api/identity/orgs", s.guard(s.handleOrgs))
}

// guard applies Meshery's authentication behaviour: a redirect, not a 401.
func (s *Server) guard(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.authenticated(r) {
			s.rejectUnauthenticated(w, r)
			return
		}
		h(w, r)
	}
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"build":           "v0.8.0",
		"commitsha":       "abc1234",
		"release_channel": "stable",
	})
}

// handleRegistry stands in for the registry family, served at both /api/registry
// and the deprecated /api/meshmodels alias. Reads are NoAuth; writes are not.
func (s *Server) handleRegistry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		if !s.authenticated(r) {
			s.rejectUnauthenticated(w, r)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"page": 0, "pageSize": 25, "totalCount": 0, "models": []any{},
	})
}

func (s *Server) handleContexts(w http.ResponseWriter, r *http.Request) {
	page, size := pageParams(r.URL.Query(), styleFor(r.URL.Path))
	total := len(s.data.Contexts)
	start, end := paginate(total, page, size)
	writeJSON(w, http.StatusOK, map[string]any{
		"page":       page,
		"pageSize":   reportedSize(size, end-start),
		"totalCount": total,
		"contexts":   s.data.Contexts[start:end],
	})
}

func (s *Server) handleConnections(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, size := pageParams(q, styleFor(r.URL.Path))

	var filtered []Connection
	kinds := q["kind"]
	for _, c := range s.data.Connections {
		if len(kinds) == 0 || contains(kinds, c.Kind) {
			filtered = append(filtered, c)
		}
	}
	start, end := paginate(len(filtered), page, size)
	writeJSON(w, http.StatusOK, map[string]any{
		"page":        page,
		"pageSize":    reportedSize(size, end-start),
		"totalCount":  len(filtered),
		"connections": filtered[start:end],
	})
}

// handleResources reproduces the cluster filter, including the difference
// between getting it wrong loudly and getting it wrong silently. A malformed
// clusterIds is a 400; an absent one binds an empty slice into
// cluster_id IN (?), matches no rows, and still answers 200.
func (s *Server) handleResources(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, size := pageParams(q, styleFor(r.URL.Path))
	ids, filter := parseClusterIDs(q)

	if filter == clusterFilterMalformed {
		writeError(w, http.StatusBadRequest, "unable to parse request body")
		return
	}

	var filtered []Resource
	if filter == clusterFilterPresent {
		for _, res := range s.data.Resources {
			if !contains(ids, res.ClusterID) {
				continue
			}
			if kinds := q["kind"]; len(kinds) > 0 && !contains(kinds, res.Kind) {
				continue
			}
			if ns := q["namespace"]; len(ns) > 0 && !contains(ns, res.Metadata.Namespace) {
				continue
			}
			filtered = append(filtered, res)
		}
	}
	// An absent filter leaves filtered nil, which is the whole point.

	if q.Get("asDesign") == "true" {
		s.writeAsDesign(w, page, size, filtered)
		return
	}

	start, end := paginate(len(filtered), page, size)
	writeJSON(w, http.StatusOK, map[string]any{
		"page":       page,
		"pageSize":   reportedSize(size, end-start),
		"totalCount": len(filtered),
		"resources":  filtered[start:end],
		// design is on the wire either way. MeshSyncResourcesAPIResponse has no
		// omitempty on it, so the key is always there and carries an empty
		// PatternFile when asDesign was not asked for. Confirmed live: the key
		// set is identical with and without the parameter.
		"design": emptyDesign(),
	})
}

// emptyDesign is the zero PatternFile Meshery serves when no design was built.
func emptyDesign() map[string]any {
	return map[string]any{
		"id":            "00000000-0000-0000-0000-000000000000",
		"name":          "",
		"schemaVersion": "",
		"version":       "",
		"components":    []any{},
		"relationships": []any{},
	}
}

// writeAsDesign reproduces the asDesign path: the flat resource list is cleared
// and a design carrying components and relationships is returned instead.
//
// Meshery's published v0.9 REST API reference puts it as "asDesign is a boolean
// value. If true then the response is returned as a design and resources are
// omitted". It is absent from docs/data/openapi.yml, the repository's only
// machine-readable spec, so a test is the only thing pinning it.
//
// The design is built from one page, not from the whole result. The handler
// applies Limit and Offset to the query before running it and hands the rows it
// got to the converter (server/handlers/meshsync_handler.go:242-360), so a
// caller asking for a graph of a large cluster on default paging gets a graph of
// the first 25 objects while totalCount reports the whole cluster. Rendering
// every row here instead would hide the truncation that costs a client its
// edges.
func (s *Server) writeAsDesign(w http.ResponseWriter, page, size int, res []Resource) {
	total := len(res)
	start, end := paginate(total, page, size)
	pageRows := res[start:end]

	components := make([]map[string]any, 0, len(pageRows))
	for _, r := range pageRows {
		components = append(components, map[string]any{
			"id":          r.ID,
			"displayName": r.Metadata.Name,
			"component":   map[string]any{"kind": r.Kind, "version": r.APIVersion},
			"model":       map[string]any{"name": "kubernetes"},
		})
	}
	relationships := []map[string]any{}
	if len(components) > 1 {
		relationships = append(relationships, map[string]any{
			"id": "e1", "kind": "hierarchical", "subType": "parent", "type": "non-binding",
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"page": page,
		// The echoed size is the limit when one was applied, and the row count
		// when the caller asked for everything.
		"pageSize": reportedSize(size, len(pageRows)),
		// The count is over every matching row, not over the page the design was
		// built from. The gap between the two is the truncation.
		"totalCount": total,
		"resources":  []Resource{}, // emptied, as the real handler does
		"design": map[string]any{
			"id":            "9f1c0b7a-0000-4000-8000-000000000001",
			"name":          "cluster",
			"schemaVersion": "designs.meshery.io/v1beta1",
			"version":       "0.0.1",
			"metadata":      map[string]any{},
			"preferences":   map[string]any{},
			"components":    components,
			"relationships": relationships,
		},
	})
}

// handleSummary requires a repeated singular clusterId and answers 400 without
// one, unlike its sibling above which takes a JSON array under clusterIds.
func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	clusterIDs := q["clusterId"]
	if len(clusterIDs) == 0 {
		writeError(w, http.StatusBadRequest, "clusterIds is required")
		return
	}
	// The guard is a presence check, so an empty or unknown value passes it and
	// then matches nothing. Repeated parameters are an IN list. Measured live:
	// clusterId=all is not special, it is just a value nothing matches.
	namespaceScope := q["namespace"]

	// The kinds census is grouped by kind and model, and the namespace scope
	// narrows it. The namespaces list is deliberately not narrowed: the live
	// handler applies the namespace filter to the kinds and labels queries only,
	// so a scoped request still reports every namespace in the cluster.
	type kindKey struct{ kind, model string }
	counts := map[kindKey]int64{}
	namespaces := []string{}
	seenNS := map[string]bool{}
	labels := []map[string]any{}
	seenLabel := map[string]bool{}

	for _, res := range s.data.Resources {
		if !contains(clusterIDs, res.ClusterID) {
			continue
		}
		if res.Metadata.Namespace != "" && !seenNS[res.Metadata.Namespace] {
			seenNS[res.Metadata.Namespace] = true
			namespaces = append(namespaces, res.Metadata.Namespace)
		}
		if len(namespaceScope) > 0 && !contains(namespaceScope, res.Metadata.Namespace) {
			continue
		}
		// Rows with no model are dropped by HAVING model IS NOT NULL.
		if res.Model == "" {
			continue
		}
		counts[kindKey{res.Kind, res.Model}]++
		for k, v := range res.Labels {
			if seenLabel[k+"="+v] {
				continue
			}
			seenLabel[k+"="+v] = true
			// Only key and value are selected, so the other columns of the row
			// come back as empty strings rather than being absent.
			labels = append(labels, map[string]any{
				"id": "", "unique_id": "", "kind": "", "key": k, "value": v,
			})
		}
	}

	// The kinds entries are a Go struct with no JSON tags on the server side, so
	// they arrive capitalized. A client reading "kind" gets an empty string and
	// no error.
	kinds := make([]map[string]any, 0, len(counts))
	for k, n := range counts {
		kinds = append(kinds, map[string]any{"Kind": k.kind, "Model": k.model, "Count": n})
	}
	sort.Slice(kinds, func(i, j int) bool {
		return kinds[i]["Kind"].(string) < kinds[j]["Kind"].(string)
	})

	// Each of the three is scanned into a nil slice and encoded as null when
	// nothing matched, which is what an unknown cluster id returns.
	writeJSON(w, http.StatusOK, map[string]any{
		"kinds":      nilIfEmptyMaps(kinds),
		"namespaces": nilIfEmptyStrings(namespaces),
		"labels":     nilIfEmptyMaps(labels),
	})
}

// nilIfEmptyMaps encodes an empty result as null rather than []. gorm scans into
// a nil slice when no row matches, and encoding/json renders that as null.
func nilIfEmptyMaps(in []map[string]any) any {
	if len(in) == 0 {
		return nil
	}
	return in
}

func nilIfEmptyStrings(in []string) any {
	if len(in) == 0 {
		return nil
	}
	return in
}

func (s *Server) handlePatterns(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, size := pageParams(q, styleFor(r.URL.Path))

	var filtered []Design
	search := strings.ToLower(q.Get("search"))
	for _, d := range s.data.Designs {
		if search == "" || strings.Contains(strings.ToLower(d.Name), search) {
			filtered = append(filtered, d)
		}
	}
	start, end := paginate(len(filtered), page, size)

	// The list endpoint serves patternFile as YAML. The by-ID endpoint serves
	// the same field as JSON. Reproducing only one of the two would hide the
	// trap entirely.
	listed := make([]Design, 0, end-start)
	for _, d := range filtered[start:end] {
		d.PatternFile = designFileYAML
		listed = append(listed, d)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"page":       page,
		"pageSize":   reportedSize(size, end-start),
		"totalCount": len(filtered),
		"patterns":   listed,
	})
}

func (s *Server) handlePatternByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/pattern/")
	for _, d := range s.data.Designs {
		if d.ID == id {
			writeJSON(w, http.StatusOK, d)
			return
		}
	}
	writeError(w, http.StatusNotFound, "design not found")
}

// handleEnvironments requires orgId, as the real handler does.
func (s *Server) handleEnvironments(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("orgId") == "" {
		writeError(w, http.StatusBadRequest, "orgId is required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"page": 0, "pageSize": 25, "total_count": 0, "environments": []any{},
	})
}

// handleWorkspaces requires orgId, accepting orgID as the deprecated spelling
// the real handler still honours.
func (s *Server) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("orgId") == "" && q.Get("orgID") == "" {
		writeError(w, http.StatusBadRequest, "orgId is required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"page": 0, "pageSize": 25, "total_count": 0, "workspaces": []any{},
	})
}

// handleOrgs emits both key spellings, as Meshery does here and nowhere else.
//
// Meshery: OrganizationsPage has a custom MarshalJSON
// (server/models/organization.go:31-42) that emits totalCount and total_count,
// and pageSize and page_size, so consumers reading either spelling keep working
// through the deprecation window. A client that reads only total_count works
// here and breaks on every other list endpoint.
// handleOrgs pages like the endpoint it stands in for. Its default page size is
// 10, not the 25 most endpoints use, and it echoes both spellings of the size
// and the count. Measured against a live server: pageSize and page_size both
// come back 10 when neither is asked for.
func (s *Server) handleOrgs(w http.ResponseWriter, r *http.Request) {
	page, size := pageParams(r.URL.Query(), styleFor(r.URL.Path))
	orgs := []map[string]string{{"id": s.data.OrgID, "name": "Default Org"}}
	start, end := paginate(len(orgs), page, size)
	reported := reportedSize(size, end-start)
	writeJSON(w, http.StatusOK, map[string]any{
		"page":          page,
		"pageSize":      reported,
		"page_size":     reported,
		"totalCount":    len(orgs),
		"total_count":   len(orgs),
		"organizations": orgs[start:end],
	})
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}
