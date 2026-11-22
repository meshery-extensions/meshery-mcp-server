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

// FamilyBudgets are Cloud's per-user token-bucket budgets, per route family.
var FamilyBudgets = map[RouteFamily]budget{
	FamilyClaims:       {rate: 10, burst: 150},
	FamilySchedules:    {rate: 20, burst: 200},
	FamilyLedgers:      {rate: 20, burst: 300},
	FamilyContentItems: {rate: 20, burst: 400},
	FamilyEventsWrite:  {rate: 10, burst: 300},
	FamilyEventsRead:   {rate: 20, burst: 500},
}

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
	bd, ok := FamilyBudgets[family]
	if !ok {
		return nil
	}
	key := string(family) + "|" + caller

	l.mu.Lock()
	bucket, exists := l.buckets[key]
	if !exists {
		bucket = NewBucket(bd.rate, bd.burst)
		l.buckets[key] = bucket
	}
	l.mu.Unlock()

	return bucket.Take(ctx)
}
