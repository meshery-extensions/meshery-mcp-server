package meshery

import (
	"encoding/json"
	"net/http"
	"time"
)

// Classification is the action the caller should take for a Meshery response.
type Classification int

const (
	// ClassOther is any status without a specific contract; existing behavior applies.
	ClassOther Classification = iota
	// ClassReauth (401): the token is missing, expired or revoked; re-authenticate.
	ClassReauth
	// ClassInsufficientScope (403): valid token, insufficient keychain scope;
	// show the missing scope; never re-authenticate.
	ClassInsufficientScope
	// ClassRateLimited (429): honor Retry-After, bounded retry.
	ClassRateLimited
	// ClassServiceUnavailable (503): backoff, bounded retry; no credential changes.
	ClassServiceUnavailable
)

func (c Classification) String() string {
	switch c {
	case ClassReauth:
		return "reauth"
	case ClassInsufficientScope:
		return "insufficient_scope"
	case ClassRateLimited:
		return "rate_limited"
	case ClassServiceUnavailable:
		return "service_unavailable"
	default:
		return "other"
	}
}

// ClassificationResult is the output of Classify.
type ClassificationResult struct {
	Class   Classification
	// MissingScope is set for 403 when Meshery's response identifies the
	// missing scope or key; empty otherwise.
	MissingScope string
	// RetryAfter is set for 429 (and 503) when Meshery sent a Retry-After
	// header the client should honor.
	RetryAfter time.Duration
}

// missingScopeKeys are the JSON keys Meshery may use to identify the missing
// scope or key.
var missingScopeKeys = []string{"missing_scope", "missing", "scope", "keychain", "key"}

// Classify turns a Meshery HTTP response into a classification per the status
// code contract. It never looks at the token itself; only at the status and
// body Meshery returns.
func Classify(resp *http.Response, body []byte) ClassificationResult {
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return ClassificationResult{Class: ClassReauth}
	case http.StatusForbidden:
		return ClassificationResult{Class: ClassInsufficientScope, MissingScope: missingScope(body)}
	case http.StatusTooManyRequests:
		retryAfter, _ := parseRetryAfter(resp.Header.Get("Retry-After"))
		return ClassificationResult{Class: ClassRateLimited, RetryAfter: retryAfter}
	case http.StatusServiceUnavailable:
		retryAfter, _ := parseRetryAfter(resp.Header.Get("Retry-After"))
		return ClassificationResult{Class: ClassServiceUnavailable, RetryAfter: retryAfter}
	default:
		return ClassificationResult{Class: ClassOther}
	}
}

// missingScope extracts the scope or key Meshery reports as missing from a JSON
// body. It returns "" when the body is not JSON or names none of the known keys.
func missingScope(body []byte) string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return ""
	}
	for _, key := range missingScopeKeys {
		value, ok := raw[key]
		if !ok || len(value) == 0 {
			continue
		}
		var s string
		if err := json.Unmarshal(value, &s); err == nil && s != "" {
			return s
		}
	}
	return ""
}
