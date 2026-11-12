package meshery

import (
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// DefaultBaseBackoff is the initial delay used when Meshery does not send
// Retry-After. It doubles on every retry, jittered and capped.
const DefaultBaseBackoff = 200 * time.Millisecond

// DefaultCapBackoff is the maximum backoff delay.
const DefaultCapBackoff = 8 * time.Second

// parseRetryAfter parses an HTTP Retry-After header value: either a
// delta-seconds integer or an HTTP-date. It returns ok=false when the value
// is absent or cannot be parsed.
func parseRetryAfter(value string) (time.Duration, bool) {
	if value == "" {
		return 0, false
	}
	if n, err := strconv.Atoi(value); err == nil && n >= 0 {
		return time.Duration(n) * time.Second, true
	}
	if t, err := http.ParseTime(value); err == nil {
		d := t.Sub(Clock())
		if d < 0 {
			return 0, true
		}
		return d, true
	}
	return 0, false
}

// Backoff computes the delay before attempt number `attempt` (0-based) of a
// retry: exponential (base * 2^attempt), plus jitter in [0, base), capped.
type Backoff struct {
	Base time.Duration
	Cap  time.Duration
}

// Delay returns the delay before the given retry attempt.
func (b Backoff) Delay(attempt int) time.Duration {
	delay := b.Base
	for i := 0; i < attempt; i++ {
		delay *= 2
	}
	jitter := time.Duration(rand.Float64() * float64(b.Base)) //nolint:gosec // G404: jitter for backoff does not require cryptographic randomness
	total := delay + jitter
	if total > b.Cap {
		total = b.Cap
	}
	return total
}
