package mesherytest_test

import (
	"fmt"
	"testing"

	"github.com/meshery-extensions/meshery-mcp-server/internal/mesherytest"
)

const summaryPath = "/api/system/meshsync/resources/summary"

// kindCounts reads the census into a kind to count map. It deliberately reads
// the capitalized keys a live server sends: the entries are a Go struct with no
// JSON tags, so a client reading "kind" gets an empty string and no error.
func kindCounts(t *testing.T, out map[string]any) map[string]float64 {
	t.Helper()
	got := map[string]float64{}
	raw, ok := out["kinds"].([]any)
	if !ok {
		t.Fatalf("kinds is not a list: %v", out["kinds"])
	}
	for _, e := range raw {
		entry, ok := e.(map[string]any)
		if !ok {
			t.Fatalf("kind entry is not an object: %v", e)
		}
		kind, ok := entry["Kind"].(string)
		if !ok || kind == "" {
			t.Fatalf(`kind entry has no "Kind": %v. The server struct carries no JSON tags, so the keys arrive capitalized`, entry)
		}
		if _, ok := entry["Model"].(string); !ok {
			t.Fatalf(`kind entry has no "Model": %v. The census groups by kind and model`, entry)
		}
		count, ok := entry["Count"].(float64)
		if !ok {
			t.Fatalf(`kind entry has no "Count": %v`, entry)
		}
		got[kind] = count
	}
	return got
}

// TestSummaryFiltersByTheClusterItIsGiven is the disagreeing fixture for the
// summary guard. Presence and value are two different checks: the 400 turns on
// presence alone, so a request naming a cluster that does not exist passes the
// guard and must then match nothing. A handler that counted every row would
// answer both of these identically.
func TestSummaryFiltersByTheClusterItIsGiven(t *testing.T) {
	s := mesherytest.New(t)

	real := authedGet(t, s, summaryPath, "clusterId="+s.Data().ClusterID())
	if counts := kindCounts(t, real); counts["Deployment"] != 1 {
		t.Errorf("the seeded cluster should be counted: %v", counts)
	}

	// Same shape of request, a cluster id nothing carries.
	other := authedGet(t, s, summaryPath, "clusterId=ksid-does-not-exist")
	if other["kinds"] != nil {
		t.Errorf("an unknown cluster must match nothing, got %v", other["kinds"])
	}
	if other["namespaces"] != nil {
		t.Errorf("an unknown cluster has no namespaces, got %v", other["namespaces"])
	}
}

// TestSummaryTakesRepeatedClusterIDsAsASet covers the IN semantics. The
// parameter is repeated rather than JSON-encoded here, which is the trap that
// separates this endpoint from its sibling.
func TestSummaryTakesRepeatedClusterIDsAsASet(t *testing.T) {
	s := mesherytest.New(t)
	ksid := s.Data().ClusterID()

	both := authedGet(t, s, summaryPath, "clusterId=ksid-absent&clusterId="+ksid)
	if counts := kindCounts(t, both); counts["Deployment"] != 1 {
		t.Errorf("a set containing the real cluster should still count it: %v", counts)
	}
}

// TestSummaryNamespaceScopeLeavesTheNamespaceList covers a genuine asymmetry in
// the live handler: the namespace filter is applied to the kinds and labels
// queries but not to the query that lists namespaces. A scoped request
// therefore reports fewer kinds and the same namespaces, which is the kind of
// thing only measurement finds.
func TestSummaryNamespaceScopeLeavesTheNamespaceList(t *testing.T) {
	s := mesherytest.New(t)
	ksid := s.Data().ClusterID()

	all := authedGet(t, s, summaryPath, "clusterId="+ksid)
	scoped := authedGet(t, s, summaryPath, "clusterId="+ksid+"&namespace=payments")

	allKinds, scopedKinds := kindCounts(t, all), kindCounts(t, scoped)
	// Nodes are cluster-scoped and carry no namespace, so scoping to a namespace
	// drops them from the census.
	if _, ok := allKinds["Node"]; !ok {
		t.Fatalf("unscoped census should include Node: %v", allKinds)
	}
	if _, ok := scopedKinds["Node"]; ok {
		t.Errorf("a namespace scope should drop cluster-scoped kinds: %v", scopedKinds)
	}

	// The cluster holds two namespaces, which is what makes the asymmetry
	// visible: with only one, a narrowed list and an unnarrowed one are the
	// same list.
	allNS, _ := all["namespaces"].([]any)
	scopedNS, _ := scoped["namespaces"].([]any)
	if len(allNS) < 2 {
		t.Fatalf("the fixture needs two namespaces for this to mean anything: %v", allNS)
	}
	if len(scopedNS) != len(allNS) {
		t.Errorf("the namespace list should not be narrowed by the namespace filter: %v then %v", allNS, scopedNS)
	}
}

// TestAsDesignIsBuiltFromOnePage is the truncation that costs a client its
// edges. The handler applies the limit before the query runs and converts only
// the rows it got, so the design describes one page while totalCount describes
// the cluster. Asking for a page smaller than the cluster is the only way to
// tell the two apart.
func TestAsDesignIsBuiltFromOnePage(t *testing.T) {
	s := mesherytest.New(t)
	ksid := s.Data().ClusterID()
	q := fmt.Sprintf(`clusterIds=["%s"]&asDesign=true&page=0&pagesize=2`, ksid)

	out := authedGet(t, s, resourcesPath, q)
	design, ok := out["design"].(map[string]any)
	if !ok {
		t.Fatalf("no design in the response: %v", out)
	}
	comps, ok := design["components"].([]any)
	if !ok {
		t.Fatalf("design carries no components: %v", design)
	}
	if len(comps) != 2 {
		t.Errorf("components = %d, want 2: the design is built from one page, not the whole cluster", len(comps))
	}
	// The count is over every matching row. The gap between the two is what
	// tells a caller the graph is partial.
	want := float64(len(s.Data().Resources))
	if total, _ := out["totalCount"].(float64); total != want {
		t.Errorf("totalCount = %v, want %v: the count is over the cluster, not the page", out["totalCount"], want)
	}

	// Asking for everything returns the whole graph, which is the control.
	whole := authedGet(t, s, resourcesPath, fmt.Sprintf(`clusterIds=["%s"]&asDesign=true&page=0&pagesize=all`, ksid))
	wholeComps := whole["design"].(map[string]any)["components"].([]any)
	if len(wholeComps) != len(s.Data().Resources) {
		t.Errorf("components with no limit = %d, want %d", len(wholeComps), len(s.Data().Resources))
	}
}
