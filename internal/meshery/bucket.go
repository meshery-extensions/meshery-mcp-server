package meshery

import (
	"context"
	"sync"
	"time"
)

// Bucket is a token bucket refilling at rate tokens per second, capped at
// burst. Take consumes one token, waiting (context-aware) until one is
// available. It is a client-side throttle only: Meshery's own enforcement
// (429) remains authoritative.
type Bucket struct {
	mu     sync.Mutex
	rate   float64 // tokens per second
	burst  float64
	tokens float64
	last   time.Time
}

// NewBucket returns a bucket that starts fully filled.
func NewBucket(rate, burst float64) *Bucket {
	return &Bucket{rate: rate, burst: burst, tokens: burst, last: time.Now()}
}

// Take consumes one token from the bucket, waiting for refill if needed.
// It returns immediately when the context is cancelled.
func (b *Bucket) Take(ctx context.Context) error {
	for {
		b.mu.Lock()
		now := time.Now()
		b.refill(now)
		if b.tokens >= 1 {
			b.tokens--
			b.last = now
			b.mu.Unlock()
			return nil
		}
		wait := time.Duration((1-b.tokens)/b.rate * float64(time.Second))
		b.last = now
		b.mu.Unlock()

		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
			continue
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		}
	}
}

// refill must be called with b.mu held.
func (b *Bucket) refill(now time.Time) {
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * b.rate
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
}
