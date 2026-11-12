package meshery

import (
	"context"
	"sync"
)

// budget is (tokens per second, burst tokens).
type budget struct {
	rate  float64
	burst float64
}

// FamilyBudgets are Meshery's per-user token-bucket budgets, per route family.
var FamilyBudgets = map[RouteFamily]budget{}

// Limiter holds one token bucket per (family, caller). Caller is the
// authenticated user id (client IP when there is no user).
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*Bucket // key: "family|caller"
}

// NewLimiter returns an empty limiter.
func NewLimiter() *Limiter {
	return &Limiter{buckets: make(map[string]*Bucket)}
}

// Take consumes one token from the bucket for (family, caller). Families
// without a configured budget are not client-side rate limited.
func (l *Limiter) Take(ctx context.Context, family RouteFamily, caller string) error {
	if _, ok := FamilyBudgets[family]; !ok {
		return nil
	}
	key := string(family) + "|" + caller

	l.mu.Lock()
	bucket, exists := l.buckets[key]
	if !exists {
		bucket = NewBucket(FamilyBudgets[family].rate, FamilyBudgets[family].burst)
		l.buckets[key] = bucket
	}
	l.mu.Unlock()

	return bucket.Take(ctx)
}
