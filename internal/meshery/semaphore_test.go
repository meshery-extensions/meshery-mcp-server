package meshery

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSemaphorePerToolLimit(t *testing.T) {
	s := NewSemaphore(2)

	ctx := context.Background()
	var mu sync.Mutex
	cur, peak := 0, 0

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Acquire(ctx, "tool-a"); err != nil {
				t.Errorf("Acquire failed: %v", err)
				return
			}
			mu.Lock()
			cur++
			if cur > peak {
				peak = cur
			}
			mu.Unlock()

			time.Sleep(50 * time.Millisecond)

			mu.Lock()
			cur--
			mu.Unlock()
			s.Release("tool-a")
		}()
	}
	wg.Wait()

	if peak != 2 {
		t.Errorf("peak concurrency = %d, want 2", peak)
	}
}

func TestSemaphoreToolsIndependent(t *testing.T) {
	s := NewSemaphore(1)

	ctx := context.Background()
	done := make(chan struct{})

	if err := s.Acquire(ctx, "tool-a"); err != nil {
		t.Fatal(err)
	}
	// A different tool may proceed while tool-a holds its slot.
	go func() {
		if err := s.Acquire(ctx, "tool-b"); err != nil {
			t.Errorf("Acquire for tool-b failed: %v", err)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("slot for another tool was blocked by tool-a")
	}
	s.Release("tool-b")
	s.Release("tool-a")
}

func TestSemaphoreReleaseUnheldNoop(t *testing.T) {
	s := NewSemaphore(1)
	// Releasing a slot that was never acquired must not deadlock a later acquirer.
	s.Release("tool-a")
	if err := s.Acquire(context.Background(), "tool-a"); err != nil {
		t.Fatal(err)
	}
	s.Release("tool-a")
}

func TestSemaphoreRespectsContext(t *testing.T) {
	s := NewSemaphore(0) // never allows a slot

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := s.Acquire(ctx, "tool-a")
	if err == nil {
		t.Error("expected context error, got nil")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
}
