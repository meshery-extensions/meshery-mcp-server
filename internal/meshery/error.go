// Package meshery provides the client used to call the Meshery Cloud API.
package meshery

import (
	"fmt"
	"net/http"
	"time"

	"github.com/meshery/meshkit/errors"
)

const (
	ErrReauthCode             = "replace_me"
	ErrInsufficientScopeCode  = "replace_me"
	ErrRateLimitedCode        = "replace_me"
	ErrServiceUnavailableCode = "replace_me"
	ErrNoTokenCode            = "replace_me"
)

// ErrReauth is returned when Cloud answers 401: the token is missing,
// expired or revoked. It is the only status that triggers re-authentication
// (the OAuth flow tracked in #66). It must never clear stored credentials.
func ErrReauth(underlying error) error {
	return errors.New(ErrReauthCode, errors.Alert,
		[]string{"Authentication required"},
		[]string{fmt.Sprintf("Cloud responded 401: %v", underlying)},
		[]string{"The API token is missing, expired or revoked"},
		[]string{"Sign in again to obtain a new token (Cloud OAuth, issue #66)"})
}

// ErrInsufficientScope is returned when Cloud answers 403: the token is
// valid but lacks the keychain scope or key the route requires. This is not
// an authentication failure and must not trigger sign-in.
type ErrInsufficientScope struct {
	// MissingScope is the scope or key Cloud identified as missing, when
	// Cloud's response identifies it; empty otherwise.
	MissingScope string
	Underlying   error
}

func (e *ErrInsufficientScope) Error() string {
	base := fmt.Sprintf("Cloud rejected the request (403 insufficient scope): %s", e.Underlying)
	if e.MissingScope != "" {
		return fmt.Sprintf("%s: the missing scope or key is %q. Ask your Cloud "+
			"administrator to grant it, or set MESHERY_KEYCHAIN_SCOPE to include it.", base, e.MissingScope)
	}
	return base + ". Cloud's response did not identify the missing scope or key; " +
		"verify that MESHERY_KEYCHAIN_SCOPE covers the keys this route needs."
}

func (e *ErrInsufficientScope) Unwrap() error { return e.Underlying }

// ErrRateLimited is returned when Cloud answers 429 and retries are
// exhausted. RetryAfter is the delay Cloud asked for, if any.
type ErrRateLimited struct {
	RetryAfter time.Duration
	Underlying error
}

func (e *ErrRateLimited) Error() string {
	base := fmt.Sprintf("rate limited by Cloud: %v", e.Underlying)
	if e.RetryAfter > 0 {
		return fmt.Sprintf("%s; retry after %d seconds", base, int(e.RetryAfter.Seconds()))
	}
	return base + "; retry later"
}

func (e *ErrRateLimited) Unwrap() error { return e.Underlying }

// ErrServiceUnavailable is returned when Cloud answers 503 and retries are
// exhausted. No credential changes are made on 503.
func ErrServiceUnavailable(underlying error) error {
	return errors.New(ErrServiceUnavailableCode, errors.Alert,
		[]string{"Cloud service unavailable"},
		[]string{fmt.Sprintf("Cloud responded 503: %v", underlying)},
		[]string{"The identity provider or a Cloud dependency is down"},
		[]string{"Retry later; no token or credential is cleared"})
}

// ErrNoToken is returned when no API token is configured for Cloud calls.
func ErrNoToken() error {
	return errors.New(ErrNoTokenCode, errors.Alert,
		[]string{"No API token configured"},
		[]string{"meshery.token / MESHERY_API_TOKEN is empty"},
		[]string{"Requests to Cloud require an API token"},
		[]string{"Set MESHERY_API_TOKEN or meshery.token"})
}

// ensure the status codes we classify are the ones we handle at compile time.
var _ = http.StatusTooManyRequests

// Clock is the source of time for parsing Retry-After HTTP dates.
var Clock = time.Now
