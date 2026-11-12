package meshery

import (
	"context"
	"sync"
)

// Semaphore bounds concurrent Meshery calls per tool. Each tool gets its own
// slot channel, so a single tool invocation (or a batch of them) cannot flood
// a route family.
type Semaphore struct {
	mu    sync.Mutex
	limit int
	tools map[string]chan struct{} // tool -> slot channel; used slots = len(channel)
}

// NewSemaphore returns a semaphore that allows limit concurrent calls per tool.
func NewSemaphore(limit int) *Semaphore {
	return &Semaphore{limit: limit, tools: make(map[string]chan struct{})}
}

// channelFor returns the slot channel for the tool, creating it on first use.
func (s *Semaphore) channelFor(tool string) chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.tools[tool]
	if !ok {
		ch = make(chan struct{}, s.limit)
		s.tools[tool] = ch
	}
	return ch
}

// Acquire holds one slot for the tool until Release is called.
func (s *Semaphore) Acquire(ctx context.Context, tool string) error {
	ch := s.channelFor(tool)
	select {
	case ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release frees one slot for the tool. Releasing an unheld slot is a no-op.
func (s *Semaphore) Release(tool string) {
	s.mu.Lock()
	ch, ok := s.tools[tool]
	s.mu.Unlock()
	if !ok {
		return
	}
	select {
	case <-ch:
	default:
	}
}
