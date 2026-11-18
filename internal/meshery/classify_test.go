package meshery

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestClassifyTable(t *testing.T) {
	// Fix the clock so http dates are deterministic regardless of the local time zone.
	oldClock := Clock
	fixedNow := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	Clock = func() time.Time { return fixedNow }
	defer func() { Clock = oldClock }()

	future := fixedNow.Add(10 * time.Second).Format(http.TimeFormat)
	past := fixedNow.Add(-10 * time.Second).Format(http.TimeFormat)

	tests := []struct {
		name           string
		status         int
		body           string
		retryAfter     string
		wantClass      Classification
		wantMissing    string
		wantRetryAfter time.Duration
	}{
		{name: "200 is other", status: 200, wantClass: ClassOther},
		{name: "401 is reauth", status: 401, wantClass: ClassReauth},
		{name: "403 with missing_scope", status: 403, body: `{"missing_scope":"org:admin"}`, wantClass: ClassInsufficientScope, wantMissing: "org:admin"},
		{name: "403 with missing key", status: 403, body: `{"missing":"key-123"}`, wantClass: ClassInsufficientScope, wantMissing: "key-123"},
		{name: "403 with scope field", status: 403, body: `{"scope":"org:admin"}`, wantClass: ClassInsufficientScope, wantMissing: "org:admin"},
		{name: "403 with keychain field", status: 403, body: `{"keychain":"kc-1"}`, wantClass: ClassInsufficientScope, wantMissing: "kc-1"},
		{name: "403 with key field", status: 403, body: `{"key":"k-1"}`, wantClass: ClassInsufficientScope, wantMissing: "k-1"},
		{name: "403 without identified scope", status: 403, body: `{"error":"forbidden"}`, wantClass: ClassInsufficientScope, wantMissing: ""},
		{name: "403 empty body", status: 403, wantClass: ClassInsufficientScope, wantMissing: ""},
		{name: "403 non-json body", status: 403, body: "forbidden", wantClass: ClassInsufficientScope, wantMissing: ""},
		{name: "429 delta seconds", status: 429, retryAfter: "5", wantClass: ClassRateLimited, wantRetryAfter: 5 * time.Second},
		{name: "429 http date future", status: 429, retryAfter: future, wantClass: ClassRateLimited, wantRetryAfter: 10 * time.Second},
		{name: "429 http date past", status: 429, retryAfter: past, wantClass: ClassRateLimited, wantRetryAfter: 0},
		{name: "429 garbage header", status: 429, retryAfter: "not-a-date", wantClass: ClassRateLimited, wantRetryAfter: 0},
		{name: "429 no header", status: 429, wantClass: ClassRateLimited, wantRetryAfter: 0},
		{name: "503", status: 503, wantClass: ClassServiceUnavailable},
		{name: "503 with retry after", status: 503, retryAfter: "3", wantClass: ClassServiceUnavailable, wantRetryAfter: 3 * time.Second},
		{name: "500 is other", status: 500, wantClass: ClassOther},
		{name: "404 is other", status: 404, wantClass: ClassOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.status,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}
			if tt.retryAfter != "" {
				resp.Header.Set("Retry-After", tt.retryAfter)
			}
			got := Classify(resp, []byte(tt.body))

			if got.Class != tt.wantClass {
				t.Errorf("Class = %v, want %v", got.Class, tt.wantClass)
			}
			if got.MissingScope != tt.wantMissing {
				t.Errorf("MissingScope = %q, want %q", got.MissingScope, tt.wantMissing)
			}
			if tt.wantRetryAfter > 0 {
				// http.date vs now() races by a few ms; allow a second of slack.
				if got.RetryAfter < tt.wantRetryAfter-time.Second || got.RetryAfter > tt.wantRetryAfter+time.Second {
					t.Errorf("RetryAfter = %v, want ~%v", got.RetryAfter, tt.wantRetryAfter)
				}
			} else if got.RetryAfter != 0 {
				t.Errorf("RetryAfter = %v, want 0", got.RetryAfter)
			}
		})
	}
}

func TestErrInsufficientScopeMessage(t *testing.T) {
	underlying := &errString{`HTTP 403 {"error":"forbidden"}`}
	identified := &ErrInsufficientScope{MissingScope: "org:admin", Underlying: underlying}
	if !strings.Contains(identified.Error(), "org:admin") {
		t.Errorf("identified scope missing from message: %s", identified.Error())
	}
	if !strings.Contains(identified.Error(), "403") {
		t.Errorf("403 missing from message: %s", identified.Error())
	}

	unidentified := &ErrInsufficientScope{MissingScope: "", Underlying: underlying}
	if !strings.Contains(unidentified.Error(), "did not identify") {
		t.Errorf("expected 'did not identify' wording: %s", unidentified.Error())
	}
}

type errString struct{ s string }

func (e *errString) Error() string { return e.s }
