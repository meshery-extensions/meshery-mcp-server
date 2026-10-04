package resources

import (
	"testing"
)

// ✅ Test that all resources are listed
func TestListResources(t *testing.T) {
	resources := ListResources()

	expected := []string{
		"meshery://connections",
		"meshery://providers",
		"meshery://adapters",
		"meshery://health",
		"meshery://environments",
	}

	if len(resources) != len(expected) {
		t.Fatalf("expected %d resources, got %d", len(expected), len(resources))
	}

	for _, exp := range expected {
		found := false
		for _, r := range resources {
			if r == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected resource %s not found in list", exp)
		}
	}
}

// ✅ Test valid resource reads
func TestReadResource_Valid(t *testing.T) {
	uris := []string{
		"meshery://connections",
		"meshery://providers",
		"meshery://adapters",
		"meshery://health",
		"meshery://environments",
	}

	for _, uri := range uris {
		t.Run(uri, func(t *testing.T) {
			data, err := ReadResource(uri)
			if err != nil {
				t.Errorf("expected no error for %s, got %v", uri, err)
			}

			if data == nil {
				t.Errorf("expected data for %s, got nil", uri)
			}
		})
	}
}

// ❌ Test invalid resource
func TestReadResource_Invalid(t *testing.T) {
	_, err := ReadResource("meshery://invalid")

	if err == nil {
		t.Error("expected error for invalid URI, got nil")
	}
}

// 🧪 Test empty / edge case handling (optional but good)
func TestReadResource_EmptyURI(t *testing.T) {
	_, err := ReadResource("")

	if err == nil {
		t.Error("expected error for empty URI, got nil")
	}
}
