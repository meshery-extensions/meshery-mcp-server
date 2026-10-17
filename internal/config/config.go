// Package config loads and validates meshery-mcp-server configuration.
//
// Precedence, lowest to highest: defaults < config file < environment
// variables < command-line flags. Secrets (the Meshery token and the HTTP
// access key) are the exception: they are read from the environment only.
package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/viper"
)

const (
	keyMesheryURL    = "meshery.url"
	keyTransportMode = "transport.mode"
	keyTransportHost = "transport.host"
	keyTransportPort = "transport.port"
	keyLogLevel      = "log.level"
	keyLogFormat     = "log.format"

	envMesheryURL    = "MESHERY_SERVER_URL"
	envMesheryToken  = "MESHERY_API_TOKEN" //nolint:gosec // G101: env var name, not a credential
	envTransportMode = "MESHERY_MCP_TRANSPORT"
	envTransportHost = "MESHERY_MCP_HOST"
	envTransportPort = "MESHERY_MCP_PORT"
	envAccessKey     = "MESHERY_MCP_ACCESS_KEY" //nolint:gosec // G101: env var name, not a credential
	envLogLevel      = "LOG_LEVEL"
	envLogFormat     = "LOG_FORMAT"

	defaultMesheryURL    = "http://localhost:9081"
	defaultTransportMode = TransportStdio
	defaultTransportHost = "127.0.0.1"
	defaultTransportPort = 8080
	defaultLogLevel      = LogLevelInfo
	defaultLogFormat     = LogFormatText

	// MinAccessKeyLength is the shortest HTTP access key accepted: 32
	// characters, as produced by `openssl rand -hex 16` or longer.
	MinAccessKeyLength = 32
)

// fileTokenKeys and fileAccessKeyKeys are config file keys that must never
// hold a secret. meshery.token was the documented location before the token
// became env-only. Viper lowercases keys, so transport.accessKey is matched
// by transport.accesskey.
var (
	fileTokenKeys     = []string{"meshery.token", "auth.token"}
	fileAccessKeyKeys = []string{"transport.access_key", "transport.accesskey"}
)

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

const (
	// LogFormatText is human-readable key=value lines, for development.
	LogFormatText = "text"
	// LogFormatJSON is one JSON object per line, for log collectors.
	LogFormatJSON = "json"
)

// TransportMode is how MCP clients reach the server.
type TransportMode string

const (
	// TransportStdio serves the one client that started the server, over
	// stdin/stdout.
	TransportStdio TransportMode = "stdio"
	// TransportHTTP serves Streamable HTTP on a loopback address.
	TransportHTTP TransportMode = "http"
)

// Config is the fully resolved, validated configuration.
type Config struct {
	Meshery   MesheryConfig   `mapstructure:"meshery"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Transport TransportConfig `mapstructure:"transport"`
	Log       LogConfig       `mapstructure:"log"`
}

// MesheryConfig describes where the Meshery server is. It holds no identity.
type MesheryConfig struct {
	URL string `mapstructure:"url"`
}

// RedactedURL returns the URL with any password replaced by "xxxxx", for logs.
func (m MesheryConfig) RedactedURL() string {
	u, err := url.Parse(m.URL)
	if err != nil {
		return "<unparsable URL>"
	}
	return u.Redacted()
}

// AuthConfig describes who the server calls Meshery as. It is kept apart
// from MesheryConfig so that where Meshery is and whose identity is used
// can change independently.
type AuthConfig struct {
	// Token is the Meshery provider token, read only from MESHERY_API_TOKEN.
	// Empty means requests carry no token, as with Meshery's Local provider.
	Token Secret `mapstructure:"-"`
}

// TransportConfig describes how MCP clients reach the server. Host, Port and
// AccessKey apply to TransportHTTP only.
type TransportConfig struct {
	Mode TransportMode `mapstructure:"mode"`
	Host string        `mapstructure:"host"`
	Port int           `mapstructure:"port"`
	// AccessKey is what HTTP clients must send as a bearer token. It is read
	// from MESHERY_MCP_ACCESS_KEY only, and only for TransportHTTP.
	AccessKey Secret `mapstructure:"-"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string `mapstructure:"level"`  // one of the LogLevel* constants
	Format string `mapstructure:"format"` // one of the LogFormat* constants
}

// Flags holds command-line overrides. A nil field was not given on the
// command line. Flags take precedence over environment variables.
type Flags struct {
	Transport *string
	Host      *string
	Port      *int
}

// Load builds the configuration. configPath is optional; an empty string
// means no file is read.
func Load(configPath string, flags Flags) (*Config, error) {
	v := viper.New()
	v.SetDefault(keyMesheryURL, defaultMesheryURL)
	v.SetDefault(keyTransportMode, string(defaultTransportMode))
	v.SetDefault(keyTransportHost, defaultTransportHost)
	v.SetDefault(keyTransportPort, defaultTransportPort)
	v.SetDefault(keyLogLevel, defaultLogLevel)
	v.SetDefault(keyLogFormat, defaultLogFormat)

	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, ErrReadConfigFile(err, configPath)
		}
		for _, key := range fileTokenKeys {
			if v.IsSet(key) {
				return nil, ErrTokenInConfigFile(key)
			}
		}
		for _, key := range fileAccessKeyKeys {
			if v.IsSet(key) {
				return nil, ErrAccessKeyInConfigFile(key)
			}
		}
	}

	// MustBindEnv only fails when called without a key, which would be a bug
	// here, not something a user can fix; panicking is appropriate.
	v.MustBindEnv(keyMesheryURL, envMesheryURL)
	v.MustBindEnv(keyTransportMode, envTransportMode)
	v.MustBindEnv(keyTransportHost, envTransportHost)
	v.MustBindEnv(keyTransportPort, envTransportPort)
	v.MustBindEnv(keyLogLevel, envLogLevel)
	v.MustBindEnv(keyLogFormat, envLogFormat)

	// Set has the highest precedence in Viper, above environment variables.
	if flags.Transport != nil {
		v.Set(keyTransportMode, *flags.Transport)
	}
	if flags.Host != nil {
		v.Set(keyTransportHost, *flags.Host)
	}
	if flags.Port != nil {
		v.Set(keyTransportPort, *flags.Port)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, ErrDecodeConfig(err)
	}
	cfg.Auth.Token = Secret(os.Getenv(envMesheryToken))
	cfg.Transport.Mode = TransportMode(strings.ToLower(strings.TrimSpace(string(cfg.Transport.Mode))))
	cfg.Transport.Host = strings.ToLower(strings.TrimSpace(cfg.Transport.Host))
	if cfg.Transport.Mode == TransportHTTP {
		cfg.Transport.AccessKey = Secret(os.Getenv(envAccessKey))
	}
	cfg.Log.Level = strings.ToLower(strings.TrimSpace(cfg.Log.Level))
	cfg.Log.Format = strings.ToLower(strings.TrimSpace(cfg.Log.Format))

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// isLoopback reports whether host only accepts connections from this machine.
// Remote HTTP needs MCP authorization (OAuth), which is not implemented, so
// the server must never be reachable from another machine. host is already
// lowercase.
func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (c *Config) validate() error {
	u, err := url.Parse(c.Meshery.URL)
	if err != nil {
		return ErrInvalidMesheryURL("The URL could not be parsed")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidMesheryURL(fmt.Sprintf("Scheme of %v must be http or https", u.Redacted()))
	}
	if u.Host == "" {
		return ErrInvalidMesheryURL(fmt.Sprintf("URL %v has no host", u.Redacted()))
	}

	switch c.Transport.Mode {
	case TransportStdio:
	case TransportHTTP:
		if !isLoopback(c.Transport.Host) {
			return ErrNonLoopbackHost(c.Transport.Host)
		}
		if c.Transport.Port < 1 || c.Transport.Port > 65535 {
			return ErrInvalidPort(c.Transport.Port)
		}
		// Loopback stops other machines, not other users or processes on this
		// one, so every HTTP request must carry the access key.
		if len(c.Transport.AccessKey) < MinAccessKeyLength {
			return ErrAccessKeyRequired()
		}
	default:
		return ErrInvalidTransport(c.Transport.Mode)
	}

	switch c.Log.Level {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
	default:
		return ErrInvalidLogLevel(c.Log.Level)
	}

	switch c.Log.Format {
	case LogFormatText, LogFormatJSON:
	default:
		return ErrInvalidLogFormat(c.Log.Format)
	}
	return nil
}
