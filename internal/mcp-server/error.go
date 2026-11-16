package mcpserver

import (
	"fmt"

	"github.com/meshery/meshkit/errors"
)

const (
	ErrListenCode       = "meshery-mcp-server-1021"
	ErrShutdownHTTPCode = "meshery-mcp-server-1022"
)

func ErrListen(err error, addr string) error {
	return errors.New(ErrListenCode, errors.Alert,
		[]string{"Unable to listen for HTTP connections"},
		[]string{fmt.Sprintf("Listening on %s failed: %v", addr, err)},
		[]string{"Another process is using the port, or the address is not available on this machine"},
		[]string{"Pick a free port with --port / MESHERY_MCP_PORT, or stop the process using it"})
}

func ErrShutdownHTTP(err error) error {
	return errors.New(ErrShutdownHTTPCode, errors.Alert,
		[]string{"HTTP server did not shut down cleanly"},
		[]string{fmt.Sprintf("Shutting down the HTTP server failed: %v", err)},
		[]string{"Requests were still running when the shutdown timeout ended"},
		[]string{"No action is needed; the server closed the remaining connections. If this repeats, check for tool calls that never finish"})
}
