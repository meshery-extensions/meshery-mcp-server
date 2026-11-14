package meshery

import (
	"context"
	"testing"
	"time"
)

func TestBucketRateAndBurst(t *testing.T) {
	b := NewBucket(10, 2)

	// The burst is available immediately.
	start := time.Now()
	if err := b.Take(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := b.Take(context.Background()); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Errorf("burst takes took %v, expected immediate", elapsed)
	}

	// After the burst is drained, the next token waits roughly
	// (1 token) / (10 tokens/s) = 100ms.
	start = time.Now()
	if err := b.Take(context.Background()); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	if elapsed < 80*time.Millisecond || elapsed > 500*time.Millisecond {
		t.Errorf("refill wait = %v, expected ~100ms", elapsed)
	}

	// After waiting long enough for two tokens to refill, two takes are immediate.
	time.Sleep(250 * time.Millisecond)
	start = time.Now()
	if err := b.Take(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := b.Take(context.Background()); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Errorf("refilled takes took %v, expected immediate", elapsed)
	}
}

func TestBucketRespectsContext(t *testing.T) {
	b := NewBucket(1, 0) // empty bucket refilling 1 token/s

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := b.Take(ctx)
	if err == nil {
		t.Error("expected context error, got nil")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
}

func TestBucketConcurrentSafe(t *testing.T) {
	b := NewBucket(1000, 100)

	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			_ = b.Take(context.Background())
			done <- true
		}()
	}
	for i := 0; i < 100; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("bucket Take deadlocked")
		}
	}
}
