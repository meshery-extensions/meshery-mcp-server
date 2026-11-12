package meshery

import (
	"testing"
)

func TestMemoryStore(t *testing.T) {
	s := &MemoryStore{}

	token, ok := s.Get()
	if ok || token != "" {
		t.Errorf("empty store = (%q, %v), want (\"\", false)", token, ok)
	}

	s.Set("my-token")
	token, ok = s.Get()
	if !ok || token != "my-token" {
		t.Errorf("after set = (%q, %v), want (\"my-token\", true)", token, ok)
	}

	s.Delete()
	token, ok = s.Get()
	if ok || token != "" {
		t.Errorf("after delete = (%q, %v), want (\"\", false)", token, ok)
	}
}
