package meshery

import (
	"net/http"
	"testing"
	"time"
)

func TestParseRetryAfter(t *testing.T) {
	// Fix the clock so http dates are deterministic regardless of the local time zone.
	oldClock := Clock
	fixedNow := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	Clock = func() time.Time { return fixedNow }
	defer func() { Clock = oldClock }()

	future := fixedNow.Add(30 * time.Second).Format(http.TimeFormat)
	past := fixedNow.Add(-30 * time.Second).Format(http.TimeFormat)

	tests := []struct {
		name   string
		value  string
		want   time.Duration
		wantOK bool
	}{
		{name: "delta seconds", value: "30", want: 30 * time.Second, wantOK: true},
		{name: "zero delta seconds", value: "0", want: 0, wantOK: true},
		{name: "http date future", value: future, want: 30 * time.Second, wantOK: true},
		{name: "http date past", value: past, want: 0, wantOK: true},
		{name: "empty", value: "", want: 0, wantOK: false},
		{name: "garbage", value: "not-a-date", want: 0, wantOK: false},
		{name: "negative delta", value: "-5", want: 0, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseRetryAfter(tt.value)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("duration = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBackoff(t *testing.T) {
	b := Backoff{Base: 100 * time.Millisecond, Cap: 500 * time.Millisecond}

	// First delay is base plus jitter in [0, base).
	d0 := b.Delay(0)
	if d0 < b.Base || d0 > 2*b.Base {
		t.Errorf("Delay(0) = %v, want in [%v, 2*%v)", d0, b.Base, b.Base)
	}

	// Delays grow exponentially.
	d1 := b.Delay(1)
	if d1 <= d0 {
		t.Errorf("Delay(1) = %v, not greater than Delay(0) = %v", d1, d0)
	}

	// Cap is respected: after enough attempts the exponential term saturates.
	for attempt := 2; attempt <= 10; attempt++ {
		d := b.Delay(attempt)
		if d > b.Cap {
			t.Errorf("Delay(%d) = %v exceeds cap %v", attempt, d, b.Cap)
		}
		if d < b.Base {
			t.Errorf("Delay(%d) = %v below base %v", attempt, d, b.Base)
		}
	}

	// Jitter makes identical attempts differ (use a low attempt so the cap
	// does not flatten every delay to the same value).
	same := make(map[time.Duration]int)
	for i := 0; i < 50; i++ {
		same[b.Delay(0)]++
	}
	if len(same) < 2 {
		t.Errorf("expected jittered delays to vary, got %v", same)
	}
}
