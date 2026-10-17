package config

import (
	"fmt"

	"github.com/meshery/meshkit/errors"
)

const (
	ErrReadConfigFileCode        = "meshery-mcp-server-1010"
	ErrDecodeConfigCode          = "meshery-mcp-server-1011"
	ErrInvalidMesheryURLCode     = "meshery-mcp-server-1012"
	ErrInvalidLogLevelCode       = "meshery-mcp-server-1013"
	ErrTokenInConfigFileCode     = "meshery-mcp-server-1014" //nolint:gosec // G101: error code, not a credential
	ErrInvalidTransportCode      = "meshery-mcp-server-1015"
	ErrNonLoopbackHostCode       = "meshery-mcp-server-1016"
	ErrInvalidPortCode           = "meshery-mcp-server-1017"
	ErrAccessKeyRequiredCode     = "meshery-mcp-server-1018"
	ErrAccessKeyInConfigFileCode = "meshery-mcp-server-1019"
	ErrInvalidLogFormatCode      = "meshery-mcp-server-1023"
)

// ErrReadConfigFile is returned when the --config file cannot be read or parsed.
func ErrReadConfigFile(err error, path string) error {
	return errors.New(ErrReadConfigFileCode, errors.Alert,
		[]string{"Unable to read the config file"},
		[]string{fmt.Sprintf("Reading %q failed: %v", path, err)},
		[]string{"The file passed with --config does not exist, is not readable, or is not valid YAML"},
		[]string{"Check the path given to --config and that the file contains valid YAML"})
}

// ErrDecodeConfig is returned when config values do not match the expected types.
func ErrDecodeConfig(err error) error {
	return errors.New(ErrDecodeConfigCode, errors.Alert,
		[]string{"Unable to decode configuration"},
		[]string{err.Error()},
		[]string{"A setting in the config file or environment has the wrong type"},
		[]string{"Check that transport.port / MESHERY_MCP_PORT is a number and every other setting is a plain string"})
}

// ErrInvalidMesheryURL is returned when MESHERY_SERVER_URL / meshery.url is unusable.
// reason must never contain the raw URL, which may embed credentials.
func ErrInvalidMesheryURL(reason string) error {
	return errors.New(ErrInvalidMesheryURLCode, errors.Alert,
		[]string{"Invalid Meshery server URL"},
		[]string{reason},
		[]string{"MESHERY_SERVER_URL or meshery.url is malformed or uses an unsupported scheme"},
		[]string{"Set MESHERY_SERVER_URL to a URL like http://localhost:9081"})
}

// ErrInvalidLogLevel is returned when LOG_LEVEL / log.level is not a supported level.
func ErrInvalidLogLevel(level string) error {
	return errors.New(ErrInvalidLogLevelCode, errors.Alert,
		[]string{"Invalid log level"},
		[]string{fmt.Sprintf("Log level %q is not supported", level)},
		[]string{"LOG_LEVEL or log.level has a typo"},
		[]string{"Set LOG_LEVEL to one of: debug, info, warn, error"})
}

// ErrTokenInConfigFile is returned when the config file contains a token.
// The token is read from MESHERY_API_TOKEN only.
func ErrTokenInConfigFile(key string) error {
	return errors.New(ErrTokenInConfigFileCode, errors.Alert,
		[]string{"Token is not allowed in the config file"},
		[]string{fmt.Sprintf("The config file sets %q", key)},
		[]string{"The token was put in the config file, which is no longer supported"},
		[]string{"Remove the token from the config file and set MESHERY_API_TOKEN instead"})
}

// ErrAccessKeyInConfigFile is returned when the config file contains the HTTP
// access key. It is read from MESHERY_MCP_ACCESS_KEY only.
func ErrAccessKeyInConfigFile(key string) error {
	return errors.New(ErrAccessKeyInConfigFileCode, errors.Alert,
		[]string{"Access key is not allowed in the config file"},
		[]string{fmt.Sprintf("The config file sets %q", key)},
		[]string{"The HTTP access key was put in the config file, which is not supported"},
		[]string{"Remove the access key from the config file and set MESHERY_MCP_ACCESS_KEY instead"})
}

// ErrInvalidTransport is returned when --transport / MESHERY_MCP_TRANSPORT /
// transport.mode is not a supported transport.
func ErrInvalidTransport(mode TransportMode) error {
	return errors.New(ErrInvalidTransportCode, errors.Alert,
		[]string{"Invalid transport"},
		[]string{fmt.Sprintf("Transport %q is not supported", mode)},
		[]string{"--transport, MESHERY_MCP_TRANSPORT or transport.mode has a typo, or names the deprecated HTTP+SSE transport"},
		[]string{"Use stdio or http. The HTTP+SSE transport is deprecated; http serves Streamable HTTP"})
}

// ErrNonLoopbackHost is returned when the http transport would listen on an
// address reachable from other machines.
func ErrNonLoopbackHost(host string) error {
	return errors.New(ErrNonLoopbackHostCode, errors.Alert,
		[]string{"HTTP transport must listen on a loopback address"},
		[]string{fmt.Sprintf("Host %q is not a loopback address", host)},
		[]string{"Remote access needs MCP authorization (OAuth), which this server does not implement yet"},
		[]string{"Set --host / MESHERY_MCP_HOST to 127.0.0.1, ::1 or localhost"})
}

// ErrInvalidPort is returned when the http transport port is out of range.
func ErrInvalidPort(port int) error {
	return errors.New(ErrInvalidPortCode, errors.Alert,
		[]string{"Invalid port"},
		[]string{fmt.Sprintf("Port %d is outside 1-65535", port)},
		[]string{"--port, MESHERY_MCP_PORT or transport.port is out of range"},
		[]string{"Set --port / MESHERY_MCP_PORT to a number between 1 and 65535"})
}

// ErrAccessKeyRequired is returned when the http transport has no access key,
// or one that is too short to resist guessing.
func ErrAccessKeyRequired() error {
	return errors.New(ErrAccessKeyRequiredCode, errors.Alert,
		[]string{"HTTP access key is required"},
		[]string{fmt.Sprintf("The http transport needs %s with at least %d characters", envAccessKey, MinAccessKeyLength)},
		[]string{"Any process on this machine can reach a loopback address, so HTTP clients must prove they are allowed to call the server"},
		[]string{fmt.Sprintf("Set %s to a random value, for example the output of: openssl rand -hex 32", envAccessKey)})
}

// ErrInvalidLogFormat is returned when LOG_FORMAT / log.format is not a supported format.
func ErrInvalidLogFormat(format string) error {
	return errors.New(ErrInvalidLogFormatCode, errors.Alert,
		[]string{"Invalid log format"},
		[]string{fmt.Sprintf("Log format %q is not supported", format)},
		[]string{"LOG_FORMAT or log.format has a typo"},
		[]string{"Set LOG_FORMAT to text or json"})
}
