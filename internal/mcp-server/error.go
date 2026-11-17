package mcpserver

import (
	"fmt"

	"github.com/meshery/meshkit/errors"
)

const (
	ErrListenCode       = "meshery-mcp-server-1020"
	ErrNoAccessKeyCode  = "meshery-mcp-server-1021"
	ErrShutdownHTTPCode = "meshery-mcp-server-1022"
)

// ErrListen is returned when the HTTP transport cannot listen on its address.
func ErrListen(err error, addr string) error {
	return errors.New(ErrListenCode, errors.Alert,
		[]string{"Unable to listen for HTTP connections"},
		[]string{fmt.Sprintf("Listening on %s failed: %v", addr, err)},
		[]string{"Another process is using the port, or the address is not available on this machine"},
		[]string{"Pick a free port with --port / MESHERY_MCP_PORT, or stop the process using it"})
}

// ErrNoAccessKey is returned when RunHTTP is called without an access key.
// Config validation normally catches this first.
func ErrNoAccessKey() error {
	return errors.New(ErrNoAccessKeyCode, errors.Alert,
		[]string{"HTTP access key is required"},
		[]string{"RunHTTP was called with an empty access key"},
		[]string{"The HTTP transport was started without MESHERY_MCP_ACCESS_KEY"},
		[]string{"Set MESHERY_MCP_ACCESS_KEY to a random value of at least 32 characters"})
}

// ErrShutdownHTTP is returned when in-flight HTTP requests do not finish
// within the shutdown timeout; the server is then closed forcibly.
func ErrShutdownHTTP(err error) error {
	return errors.New(ErrShutdownHTTPCode, errors.Alert,
		[]string{"HTTP server did not shut down cleanly"},
		[]string{fmt.Sprintf("Shutting down the HTTP server failed: %v", err)},
		[]string{"Requests were still running when the shutdown timeout ended"},
		[]string{"No action is needed; the server closed the remaining connections. If this repeats, check for tool calls that never finish"})
}
