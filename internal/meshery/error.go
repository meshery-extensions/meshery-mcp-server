package meshery

import (
	"fmt"
	"time"

	"github.com/meshery/meshkit/errors"
)

var (
	ErrInvalidBaseURLCode    = "replace_me"
	ErrInvalidTimeoutCode    = "replace_me"
	ErrInvalidRetryCountCode = "replace_me"
	ErrHTTPRequestCode       = "replace_me"
	ErrAPIResponseCode       = "replace_me"
)

// ErrInvalidBaseURL returns a MeshKit Error when base URL parsing or scheme validation fails.
func ErrInvalidBaseURL(err error) error {
	return errors.New(
		ErrInvalidBaseURLCode,
		errors.Alert,
		[]string{"Invalid base URL configuration"},
		[]string{err.Error()},
		[]string{"Base URL string is malformed or missing scheme/host"},
		[]string{"Ensure BaseURL is a valid http or https URL (e.g. http://localhost:9081)"},
	)
}

// ErrInvalidTimeout returns a MeshKit Error when client timeout configuration is non-positive.
func ErrInvalidTimeout(timeout time.Duration) error {
	return errors.New(
		ErrInvalidTimeoutCode,
		errors.Alert,
		[]string{fmt.Sprintf("Invalid HTTP client timeout: %v", timeout)},
		[]string{"Timeout duration must be greater than zero"},
		[]string{"Timeout parameter in Config was set to zero or negative duration"},
		[]string{"Set Timeout to a positive duration (e.g. 5s, 10s)"},
	)
}

// ErrInvalidRetryCount returns a MeshKit Error when retry count configuration is negative.
func ErrInvalidRetryCount(retryCount int) error {
	return errors.New(
		ErrInvalidRetryCountCode,
		errors.Alert,
		[]string{fmt.Sprintf("Invalid retry count: %d", retryCount)},
		[]string{"Retry count cannot be negative"},
		[]string{"RetryCount parameter in Config was set below 0"},
		[]string{"Set RetryCount to 0 or positive integer"},
	)
}

// ErrHTTPRequest returns a MeshKit Error when network transport or HTTP request execution fails.
func ErrHTTPRequest(err error, method, reqURL string) error {
	return errors.New(
		ErrHTTPRequestCode,
		errors.Alert,
		[]string{fmt.Sprintf("HTTP request failed [%s %s]", method, reqURL)},
		[]string{err.Error()},
		[]string{"Network connectivity failure or endpoint host unreachable"},
		[]string{"Verify Meshery Server instance is running and network endpoint is accessible"},
	)
}

// ErrAPIResponse returns a MeshKit Error when Meshery Server returns an HTTP error status code (4xx/5xx).
func ErrAPIResponse(statusCode int, method, reqURL, msg string) error {
	return errors.New(
		ErrAPIResponseCode,
		errors.Alert,
		[]string{fmt.Sprintf("meshery API [%s %s] failed (%d): %s", method, reqURL, statusCode, msg)},
		[]string{msg},
		[]string{fmt.Sprintf("Meshery Server returned HTTP error status %d", statusCode)},
		[]string{"Check server logs and ensure resource exists or client token has valid permissions"},
	)
}
