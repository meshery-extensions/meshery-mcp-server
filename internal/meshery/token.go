package meshery

import "sync"

// TokenStore holds the current API token. Implementations may back the store
// with memory, disk, a keyring or a session store. Revoke empties every
// store regardless of the server's response.
type TokenStore interface {
	// Get returns the current token and whether one is present.
	Get() (token string, ok bool)
	// Set stores a token.
	Set(token string)
	// Delete removes the token, e.g. after revocation.
	Delete()
}

// MemoryStore keeps the token in process memory.
type MemoryStore struct {
	mu    sync.Mutex
	token string
}

func (s *MemoryStore) Get() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token, s.token != ""
}

func (s *MemoryStore) Set(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = token
}

func (s *MemoryStore) Delete() {
	s.Set("")
}
