package mesherytest

import "fmt"

// Data is the fixture set the fake serves. Field names and JSON tags follow
// Meshery's wire format rather than anything convenient.
type Data struct {
	Contexts    []Context
	Connections []Connection
	Designs     []Design
	Resources   []Resource
	OrgID       string
}

// Context mirrors an entry from GET /api/system/kubernetes/contexts.
//
// The three identifiers here are distinct and not interchangeable:
// KubernetesServerID is what MeshSync keys resources on, ConnectionID
// addresses the connection record, and ID is the deployment target passed as
// ?contexts=.
type Context struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Server             string `json:"server"`
	Version            string `json:"version"`
	ConnectionID       string `json:"connectionId"`
	KubernetesServerID string `json:"kubernetesServerId"`
}

// Connection mirrors an entry from GET /api/integrations/connections.
type Connection struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

// DesignFileYAML is the YAML encoding the list endpoint serves, exposed so a
// test can assert against the exact bytes a client would receive there.
func DesignFileYAML() string { return designFileYAML }

// Design mirrors an entry from GET /api/pattern.
//
// PatternFile is a JSON *string* on current Meshery, not a nested object, and
// it is served under the camelCase key. Decoding it as an object, or looking
// only for the older pattern_file spelling, yields an empty design and no
// error.
type Design struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PatternFile string `json:"patternFile"`
}

// Resource mirrors a MeshSync-discovered object from
// GET /api/system/meshsync/resources.
type Resource struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	APIVersion string   `json:"apiVersion"`
	ClusterID  string   `json:"cluster_id"`
	Metadata   Metadata `json:"metadata"`

	// Model is the registry model a resource belongs to. The summary census
	// groups by kind and model and drops rows where it is null, so a resource
	// without one is counted nowhere.
	Model string `json:"model,omitempty"`
	// Labels are returned by the summary as distinct key and value pairs. They
	// are never returned on a resource row: the client does not ask for them,
	// which is what keeps Secret payloads off the wire.
	Labels map[string]string `json:"-"`
}

// Metadata is the subset of object metadata MeshSync returns by default.
type Metadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// The same design in the two encodings Meshery actually serves.
//
// GET /api/pattern returns patternFile as YAML and GET /api/pattern/{id}
// returns it as JSON. Verified against a Meshery Server built from master and
// run locally: six designs checked, all six YAML from the list and JSON by ID.
// SaveMesheryPattern stores the design with yaml.Marshal
// (server/models/meshery_pattern_persister.go:233), and the list path returns
// that stored form verbatim.
//
// A client that reads the design out of the list response and decodes it as
// JSON therefore fails on every design, while passing against any mock that
// serves JSON from both. This is the one behaviour here that source reading
// alone did not reveal.
const designFileJSON = `{"name":"bookinfo","schemaVersion":"designs.meshery.io/v1beta1",` +
	`"components":[` +
	`{"id":"c1","displayName":"productpage","component":{"kind":"Deployment","version":"apps/v1"}},` +
	`{"id":"c2","displayName":"tls-cert","component":{"kind":"Secret","version":"v1"}}],` +
	`"relationships":[{"id":"r1","kind":"edge","subType":"network","type":"non-binding"}]}`

const designFileYAML = `name: bookinfo
schemaVersion: designs.meshery.io/v1beta1
components:
- id: c1
  displayName: productpage
  component:
    kind: Deployment
    version: apps/v1
- id: c2
  displayName: tls-cert
  component:
    kind: Secret
    version: v1
relationships:
- id: r1
  kind: edge
  subType: network
  type: non-binding
`

// SeedData returns a small dataset: one cluster, one connection, two designs
// and four discovered resources, one of which is a Secret so that a Secret
// exclusion guarantee has something to exclude.
func SeedData() *Data {
	const ksid = "ksid-9c2e"
	return &Data{
		OrgID: "org-1a2b",
		Contexts: []Context{{
			ID:                 "ctx-7f3a",
			Name:               "minikube",
			Server:             "https://127.0.0.1:6443",
			Version:            "v1.31.0",
			ConnectionID:       "conn-42b1",
			KubernetesServerID: ksid,
		}},
		Connections: []Connection{{
			ID:     "conn-42b1",
			Name:   "minikube",
			Kind:   "kubernetes",
			Status: "connected",
		}},
		Designs: seedDesigns(),
		Resources: []Resource{
			{ID: "r1", Kind: "Deployment", APIVersion: "apps/v1", ClusterID: ksid, Model: "kubernetes",
				Metadata: Metadata{Name: "productpage", Namespace: "payments"},
				Labels:   map[string]string{"app": "productpage"}},
			{ID: "r2", Kind: "Pod", APIVersion: "v1", ClusterID: ksid, Model: "kubernetes",
				Metadata: Metadata{Name: "productpage-7d4", Namespace: "payments"},
				Labels:   map[string]string{"app": "productpage"}},
			{ID: "r3", Kind: "Service", APIVersion: "v1", ClusterID: ksid, Model: "kubernetes",
				Metadata: Metadata{Name: "productpage-svc", Namespace: "payments"}},
			{ID: "r4", Kind: "Secret", APIVersion: "v1", ClusterID: ksid, Model: "kubernetes",
				Metadata: Metadata{Name: "db-credentials", Namespace: "payments"}},
			// A second namespace, so that scoping to one is distinguishable from
			// not scoping at all. With a single namespace the two produce the
			// same answer and a test cannot tell a working filter from a
			// missing one.
			{ID: "r7", Kind: "ConfigMap", APIVersion: "v1", ClusterID: ksid, Model: "kubernetes",
				Metadata: Metadata{Name: "cluster-settings", Namespace: "default"}},
			// Nodes are cluster-scoped, so they carry no namespace. A cluster
			// with none is not a cluster, and a tool that walks nodes needs
			// something to walk.
			{ID: "r5", Kind: "Node", APIVersion: "v1", ClusterID: ksid, Model: "kubernetes",
				Metadata: Metadata{Name: "minikube"}},
			{ID: "r6", Kind: "Node", APIVersion: "v1", ClusterID: ksid, Model: "kubernetes",
				Metadata: Metadata{Name: "minikube-m02"}},
		},
	}
}

// seedDesigns returns more than one default page of designs. The count matters:
// with no more rows than a default page, "no limit" and "fell back to the default" return the
// same thing, and a test cannot tell them apart.
func seedDesigns() []Design {
	designs := []Design{
		{ID: "d-1001", Name: "bookinfo", PatternFile: designFileJSON},
		{ID: "d-1002", Name: "redis-cache", PatternFile: designFileJSON},
	}
	for i := 3; i <= 30; i++ {
		designs = append(designs, Design{
			ID:          fmt.Sprintf("d-%04d", 1000+i),
			Name:        fmt.Sprintf("design-%02d", i),
			PatternFile: designFileJSON,
		})
	}
	return designs
}

// ClusterID returns the Kubernetes server ID of the seeded cluster, which is
// the value the cluster-scoped endpoints expect.
func (d *Data) ClusterID() string {
	if len(d.Contexts) == 0 {
		return ""
	}
	return d.Contexts[0].KubernetesServerID
}
