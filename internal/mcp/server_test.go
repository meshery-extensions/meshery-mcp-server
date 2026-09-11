package mcp

import "testing"

func TestServerStart(t *testing.T) {
	s := NewServer()
	res := s.Start()

	if res != "pong" {
		t.Errorf("expected pong, got %s", res)
	}
}
