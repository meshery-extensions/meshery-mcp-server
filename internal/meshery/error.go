// Package meshery provides the client used to call the Meshery Server API.
package meshery

import (
	"fmt"
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

// ErrReauth is returned when Meshery answers 401: the token is missing,
// expired or revoked. It is the only status that triggers re-authentication.
func ErrReauth(underlying error) error {
	return errors.New(ErrReauthCode, errors.Alert,
		[]string{"Authentication required"},
		[]string{fmt.Sprintf("Meshery responded 401: %v", underlying)},
		[]string{"The API token is missing, expired or revoked"},
		[]string{"Sign in again to obtain a new token"})
}

// ErrInsufficientScope is returned when Meshery answers 403: the token is
// valid but lacks the keychain scope or key the route requires. This is not
// an authentication failure and must not trigger sign-in.
type ErrInsufficientScope struct {
	// MissingScope is the scope or key Meshery identified as missing, when
	// Meshery's response identifies it; empty otherwise.
	MissingScope string
	Underlying   error
}

func (e *ErrInsufficientScope) Error() string {
	base := fmt.Sprintf("Meshery rejected the request (403 insufficient scope): %s", e.Underlying)
	if e.MissingScope != "" {
		return fmt.Sprintf("%s: the missing scope or key is %q.", base, e.MissingScope)
	}
	return base + ". Meshery's response did not identify the missing scope or key."
}

func (e *ErrInsufficientScope) Unwrap() error { return e.Underlying }

// ErrRateLimited is returned when Meshery answers 429 and retries are
// exhausted. RetryAfter is the delay Meshery asked for, if any.
type ErrRateLimited struct {
	RetryAfter time.Duration
	Underlying error
}

func (e *ErrRateLimited) Error() string {
	base := fmt.Sprintf("rate limited by Meshery: %v", e.Underlying)
	if e.RetryAfter > 0 {
		return fmt.Sprintf("%s; retry after %d seconds", base, int(e.RetryAfter.Seconds()))
	}
	return base + "; retry later"
}

func (e *ErrRateLimited) Unwrap() error { return e.Underlying }

// ErrServiceUnavailable is returned when Meshery answers 503 and retries are
// exhausted. No credential changes are made on 503.
func ErrServiceUnavailable(underlying error) error {
	return errors.New(ErrServiceUnavailableCode, errors.Alert,
		[]string{"Meshery service unavailable"},
		[]string{fmt.Sprintf("Meshery responded 503: %v", underlying)},
		[]string{"Meshery or a dependency is down"},
		[]string{"Retry later; no token or credential is cleared"})
}

// ErrNoToken is returned when no API token is configured for Meshery calls.
func ErrNoToken() error {
	return errors.New(ErrNoTokenCode, errors.Alert,
		[]string{"No API token configured"},
		[]string{"meshery.token / MESHERY_API_TOKEN is empty"},
		[]string{"Requests to Meshery require an API token"},
		[]string{"Set MESHERY_API_TOKEN or meshery.token"})
}

// Clock is the source of time for parsing Retry-After HTTP dates.
var Clock = time.Now
