package config

import (
	"fmt"

	"github.com/meshery/meshkit/errors"
)

const (
	ErrReadConfigFileCode            = "meshery-mcp-server-1010"
	ErrDecodeConfigCode              = "meshery-mcp-server-1011"
	ErrInvalidMesheryURLCode         = "meshery-mcp-server-1012"
	ErrInvalidLogLevelCode           = "meshery-mcp-server-1013"
	ErrTokenInConfigFileCode         = "meshery-mcp-server-1014" //nolint:gosec // G101: error code, not a credential
	ErrInvalidTransportCode          = "meshery-mcp-server-1015"
	ErrNonLoopbackHostCode           = "meshery-mcp-server-1016"
	ErrInvalidPortCode               = "meshery-mcp-server-1017"
	ErrInvalidLogFormatCode          = "meshery-mcp-server-1018"
	ErrTokenNotAllowedForHTTPCode    = "meshery-mcp-server-1019" //nolint:gosec // G101: error code, not a credential
	ErrHTTPAuthorizationRequiredCode = "meshery-mcp-server-1020"
	ErrTokenWithLocalProviderCode    = "meshery-mcp-server-1028" //nolint:gosec // G101: error code, not a credential
)

func ErrReadConfigFile(err error, path string) error {
	return errors.New(ErrReadConfigFileCode, errors.Alert,
		[]string{"Unable to read the config file"},
		[]string{fmt.Sprintf("Reading %q failed: %v", path, err)},
		[]string{"The file passed with --config does not exist, is not readable, or is not valid YAML"},
		[]string{"Check the path given to --config and that the file contains valid YAML"})
}

func ErrDecodeConfig(err error) error {
	return errors.New(ErrDecodeConfigCode, errors.Alert,
		[]string{"Unable to decode configuration"},
		[]string{err.Error()},
		[]string{"A setting in the config file or environment has the wrong type"},
		[]string{"Check that transport.http.port / MESHERY_MCP_PORT is a number and every other setting is a plain string"})
}

// reason must never contain the raw URL, which may embed credentials.
func ErrInvalidMesheryURL(reason string) error {
	return errors.New(ErrInvalidMesheryURLCode, errors.Alert,
		[]string{"Invalid Meshery server URL"},
		[]string{reason},
		[]string{"MESHERY_SERVER_URL or meshery.url is malformed or uses an unsupported scheme"},
		[]string{"Set MESHERY_SERVER_URL to a URL like http://localhost:9081"})
}

func ErrInvalidLogLevel(level string) error {
	return errors.New(ErrInvalidLogLevelCode, errors.Alert,
		[]string{"Invalid log level"},
		[]string{fmt.Sprintf("Log level %q is not supported", level)},
		[]string{"LOG_LEVEL or log.level has a typo"},
		[]string{"Set LOG_LEVEL to one of: debug, info, warn, error"})
}

func ErrTokenInConfigFile(key string) error {
	return errors.New(ErrTokenInConfigFileCode, errors.Alert,
		[]string{"Token is not allowed in the config file"},
		[]string{fmt.Sprintf("The config file sets %q", key)},
		[]string{"The token was put in the config file, which is not supported: config files get copied, shared and committed"},
		[]string{"Remove the token from the config file and set MESHERY_API_TOKEN instead"})
}

func ErrInvalidTransport(mode TransportMode) error {
	return errors.New(ErrInvalidTransportCode, errors.Alert,
		[]string{"Invalid transport"},
		[]string{fmt.Sprintf("Transport %q is not supported", mode)},
		[]string{"--transport, MESHERY_MCP_TRANSPORT or transport.mode has a typo, or names the deprecated HTTP+SSE transport"},
		[]string{"Use stdio or http. The HTTP+SSE transport is deprecated; http serves Streamable HTTP"})
}

func ErrNonLoopbackHost(host string) error {
	return errors.New(ErrNonLoopbackHostCode, errors.Alert,
		[]string{"HTTP transport must listen on a loopback address"},
		[]string{fmt.Sprintf("Host %q is not a loopback address", host)},
		[]string{"The http transport listens on loopback addresses only"},
		[]string{"Set --host / MESHERY_MCP_HOST to 127.0.0.1, ::1 or localhost"})
}

func ErrInvalidPort(port int) error {
	return errors.New(ErrInvalidPortCode, errors.Alert,
		[]string{"Invalid port"},
		[]string{fmt.Sprintf("Port %d is outside 1-65535", port)},
		[]string{"--port, MESHERY_MCP_PORT or transport.http.port is out of range"},
		[]string{"Set --port / MESHERY_MCP_PORT to a number between 1 and 65535"})
}

func ErrInvalidLogFormat(format string) error {
	return errors.New(ErrInvalidLogFormatCode, errors.Alert,
		[]string{"Invalid log format"},
		[]string{fmt.Sprintf("Log format %q is not supported", format)},
		[]string{"LOG_FORMAT or log.format has a typo"},
		[]string{"Set LOG_FORMAT to text or json"})
}

func ErrTokenNotAllowedForHTTP() error {
	return errors.New(ErrTokenNotAllowedForHTTPCode, errors.Alert,
		[]string{"Meshery token is not allowed with the http transport"},
		[]string{fmt.Sprintf("%s is set and the transport is http", envMesheryToken)},
		[]string{"Over http, every caller would act as the owner of the token. Each HTTP client must obtain its own Meshery identity through MCP authorization (OAuth)"},
		[]string{fmt.Sprintf("Unset %s to run over http, or use the stdio transport", envMesheryToken)})
}

func ErrHTTPAuthorizationRequired() error {
	return errors.New(ErrHTTPAuthorizationRequiredCode, errors.Alert,
		[]string{"The http transport requires MCP authorization"},
		[]string{"The transport is http and MCP authorization (OAuth) is not configured"},
		[]string{"Over http, each client obtains its Meshery identity through OAuth; without it no client could call Meshery"},
		[]string{"Configure MCP authorization, or use the stdio transport and set MESHERY_API_TOKEN"})
}

func ErrTokenWithLocalProvider(provider string) error {
	return errors.New(ErrTokenWithLocalProviderCode, errors.Alert,
		[]string{"Meshery token is set with the Local provider"},
		[]string{fmt.Sprintf("%s is set and the provider (%s or transport.stdio.provider) is %q", envMesheryToken, envProvider, provider)},
		[]string{"Meshery's Local provider ignores tokens, so every request would run as the local user, not as the token's owner"},
		[]string{fmt.Sprintf("Set %s to the provider the token belongs to, for example %s, or unset %s", envProvider, DefaultRemoteProvider, envMesheryToken)})
}
