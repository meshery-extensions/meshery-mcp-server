// Package errors provides canonical MeshKit wire-format error constructors for Meshery MCP Server.
package errors

import (
	"github.com/meshery/meshkit/errors"
)

// Error codes for Meshery MCP Server
const (
	// ErrUnauthenticatedCode indicates missing or invalid Meshery Server authentication credentials.
	ErrUnauthenticatedCode = "1001-MCP"
	// ErrUpstreamFailedCode indicates upstream HTTP API connection failures to Meshery Server.
	ErrUpstreamFailedCode = "1002-MCP"
	// ErrSchemaInvalidCode indicates invalid input tool arguments failing JSON schema validation.
	ErrSchemaInvalidCode = "1003-MCP"
)

// ErrUnauthenticated creates a canonical MeshKit error for authentication failures.
func ErrUnauthenticated(err error) error {
	return errors.New(
		ErrUnauthenticatedCode,
		errors.Alert,
		[]string{"Authentication Failure"},
		[]string{"Failed to authenticate request against Meshery API provider."},
		[]string{"Authenticated Meshery context is missing, invalid, or expired."},
		[]string{"Verify valid Meshery Server authentication context and credentials."},
	)
}

// ErrUpstreamFailed creates a canonical MeshKit error for upstream API query failures.
func ErrUpstreamFailed(err error) error {
	return errors.New(
		ErrUpstreamFailedCode,
		errors.Alert,
		[]string{"Upstream Meshery Unreachable"},
		[]string{"Failed to query upstream Meshery Server endpoint."},
		[]string{"Meshery Server is offline or network partition occurred."},
		[]string{"Ensure Meshery Server is running at target URL and reachable over network."},
	)
}

// ErrSchemaInvalid creates a canonical MeshKit error for payload validation failures.
func ErrSchemaInvalid(err error) error {
	return errors.New(
		ErrSchemaInvalidCode,
		errors.Alert,
		[]string{"Schema Validation Failed"},
		[]string{"Incoming tool arguments failed JSON schema validation."},
		[]string{"Caller provided invalid parameters or schema mismatch."},
		[]string{"Validate request arguments against tool schema definition."},
	)
}
