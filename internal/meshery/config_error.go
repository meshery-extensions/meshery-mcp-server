package meshery

import (
	"fmt"
	"time"

	"github.com/meshery/meshkit/errors"
)

const (
	ErrInvalidBaseURLCode            = "replace_me"
	ErrInvalidAuthModeCode           = "replace_me"
	ErrInvalidTimeoutCode            = "replace_me"
	ErrInvalidMaxRetriesCode         = "replace_me"
	ErrInvalidMaxConcurrentCallsCode = "replace_me"
)

// ErrInvalidBaseURL is returned when the configured base URL is not a
// well-formed http(s) URL with a host. It never echoes the URL, which may
// embed credentials.
func ErrInvalidBaseURL() error {
	return errors.New(ErrInvalidBaseURLCode, errors.Alert,
		[]string{"Invalid Meshery server URL"},
		[]string{"The client base URL must be an http or https URL with a host and no embedded credentials"},
		[]string{"MESHERY_SERVER_URL or meshery.url is malformed"},
		[]string{"Set it to a URL like http://localhost:9081"})
}

// ErrInvalidAuthMode is returned for an unknown authentication mode.
func ErrInvalidAuthMode(mode string) error {
	return errors.New(ErrInvalidAuthModeCode, errors.Alert,
		[]string{"Invalid authentication mode"},
		[]string{fmt.Sprintf("auth mode %q is not supported", mode)},
		[]string{"The client supports cookie and bearer authentication only"},
		[]string{"Use cookie for Meshery Server or bearer for Meshery Cloud"})
}

// ErrInvalidTimeout is returned for a negative request timeout.
func ErrInvalidTimeout(value time.Duration) error {
	return errors.New(ErrInvalidTimeoutCode, errors.Alert,
		[]string{"Invalid request timeout"},
		[]string{fmt.Sprintf("timeout is %v, must not be negative", value)},
		[]string{"The timeout setting is negative"},
		[]string{"Set a positive duration such as 30s"})
}

// ErrInvalidMaxRetries is returned for a negative retry bound.
func ErrInvalidMaxRetries(value int) error {
	return errors.New(ErrInvalidMaxRetriesCode, errors.Alert,
		[]string{"Invalid max retries"},
		[]string{fmt.Sprintf("max retries is %d, must not be negative", value)},
		[]string{"The retry setting is negative"},
		[]string{"Set it to an integer >= 0"})
}

// ErrInvalidMaxConcurrentCalls is returned for a negative concurrency bound.
func ErrInvalidMaxConcurrentCalls(value int) error {
	return errors.New(ErrInvalidMaxConcurrentCallsCode, errors.Alert,
		[]string{"Invalid max concurrent calls"},
		[]string{fmt.Sprintf("max concurrent calls is %d, must not be negative", value)},
		[]string{"The concurrency setting is negative"},
		[]string{"Set it to an integer >= 1"})
}
