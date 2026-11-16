// Package config loads and validates the server's configuration.
//
// Precedence, lowest to highest: defaults, config file, environment
// variables, flags. The Meshery token is read from the environment only.
//
// Settings that belong to one transport live under it. stdio calls Meshery
// as the user who started the server; over http each client gets its own
// identity through MCP authorization, so a configured token is never used.
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
	keyStdioProvider = "transport.stdio.provider"
	keyHTTPHost      = "transport.http.host"
	keyHTTPPort      = "transport.http.port"
	keyLogLevel      = "log.level"
	keyLogFormat     = "log.format"

	envMesheryURL    = "MESHERY_SERVER_URL"
	envMesheryToken  = "MESHERY_API_TOKEN" //nolint:gosec // G101: env var name, not a credential
	envProvider      = "MESHERY_PROVIDER"
	envTransportMode = "MESHERY_MCP_TRANSPORT"
	envHTTPHost      = "MESHERY_MCP_HOST"
	envHTTPPort      = "MESHERY_MCP_PORT"
	envLogLevel      = "LOG_LEVEL"
	envLogFormat     = "LOG_FORMAT"

	defaultMesheryURL    = "http://localhost:9081"
	defaultTransportMode = TransportStdio
	defaultHTTPHost      = "127.0.0.1"
	defaultHTTPPort      = 8080
	defaultLogLevel      = LogLevelInfo
	defaultLogFormat     = LogFormatText
)

// The token is not listed: it has no config key and is read for stdio only.
var envBindings = []struct{ key, env string }{
	{keyMesheryURL, envMesheryURL},
	{keyTransportMode, envTransportMode},
	{keyStdioProvider, envProvider},
	{keyHTTPHost, envHTTPHost},
	{keyHTTPPort, envHTTPPort},
	{keyLogLevel, envLogLevel},
	{keyLogFormat, envLogFormat},
}

// Config file keys someone would plausibly put a token under.
var fileTokenKeys = []string{
	"meshery.token", "auth.token", "transport.stdio.token",
}

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

const (
	LogFormatText = "text"
	LogFormatJSON = "json"
)

// TransportMode is how MCP clients reach the server.
type TransportMode string

const (
	TransportStdio TransportMode = "stdio"
	TransportHTTP  TransportMode = "http"
)

type Config struct {
	Meshery   MesheryConfig   `mapstructure:"meshery"`
	Transport TransportConfig `mapstructure:"transport"`
	Log       LogConfig       `mapstructure:"log"`
}

// MesheryConfig says where the Meshery server is; identity lives in the
// transport config.
type MesheryConfig struct {
	URL string `mapstructure:"url"`
}

// RedactedURL returns the URL with any password replaced, for logs.
func (m MesheryConfig) RedactedURL() string {
	u, err := url.Parse(m.URL)
	if err != nil {
		return "<unparsable URL>"
	}
	return u.Redacted()
}

// TransportConfig holds one sub-config per transport; only Mode's is used.
type TransportConfig struct {
	Mode  TransportMode `mapstructure:"mode"`
	Stdio StdioConfig   `mapstructure:"stdio"`
	HTTP  HTTPConfig    `mapstructure:"http"`
}

// StdioConfig is the identity of the user who started the server. Meshery
// picks the provider first (unless the Meshery server pins one) and that
// provider then checks the token, so the provider is configured, never
// inferred from the token.
type StdioConfig struct {
	Provider string `mapstructure:"provider"`
	Token    Secret `mapstructure:"-"`
}

const (
	// LocalProvider has no user sessions: it needs no token and ignores one.
	LocalProvider = "Local"
	// DefaultRemoteProvider is assumed for a token when none is set
	// (cloud.meshery.io).
	DefaultRemoteProvider = "Meshery"
)

// None is Local's legacy name, which Meshery still accepts.
func isLocalProvider(name string) bool {
	return strings.EqualFold(name, LocalProvider) || strings.EqualFold(name, "None")
}

type HTTPConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Flags holds command-line overrides; nil means the flag was not given.
type Flags struct {
	Transport *string
	Host      *string
	Port      *int
}

// Load builds the configuration. An empty configPath reads no file.
func Load(configPath string, flags Flags) (*Config, error) {
	cfg, err := load(configPath, flags)
	if err != nil {
		return nil, err
	}
	// Without MCP authorization, HTTP clients would have no Meshery identity.
	if cfg.Transport.Mode == TransportHTTP {
		return nil, ErrHTTPAuthorizationRequired()
	}
	return cfg, nil
}

// load resolves and validates every transport, http included.
func load(configPath string, flags Flags) (*Config, error) {
	v := viper.New()
	v.SetDefault(keyMesheryURL, defaultMesheryURL)
	v.SetDefault(keyTransportMode, string(defaultTransportMode))
	v.SetDefault(keyStdioProvider, "") // depends on the token; see StdioConfig.Provider
	v.SetDefault(keyHTTPHost, defaultHTTPHost)
	v.SetDefault(keyHTTPPort, defaultHTTPPort)
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
	}

	// MustBindEnv panics only when given no key, a programming error.
	for _, b := range envBindings {
		v.MustBindEnv(b.key, b.env)
	}

	// Set has the highest precedence in Viper, above environment variables.
	if flags.Transport != nil {
		v.Set(keyTransportMode, *flags.Transport)
	}
	if flags.Host != nil {
		v.Set(keyHTTPHost, *flags.Host)
	}
	if flags.Port != nil {
		v.Set(keyHTTPPort, *flags.Port)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, ErrDecodeConfig(err)
	}
	cfg.Transport.Mode = TransportMode(strings.ToLower(strings.TrimSpace(string(cfg.Transport.Mode))))
	cfg.Transport.HTTP.Host = strings.ToLower(strings.TrimSpace(cfg.Transport.HTTP.Host))
	cfg.Log.Level = strings.ToLower(strings.TrimSpace(cfg.Log.Level))
	cfg.Log.Format = strings.ToLower(strings.TrimSpace(cfg.Log.Format))

	// A token set for http fails startup instead of being silently dropped.
	switch cfg.Transport.Mode {
	case TransportStdio:
		stdio := &cfg.Transport.Stdio
		stdio.Token = Secret(os.Getenv(envMesheryToken))
		stdio.Provider = strings.TrimSpace(stdio.Provider)
		if stdio.Provider == "" {
			stdio.Provider = LocalProvider
			if stdio.Token != "" {
				stdio.Provider = DefaultRemoteProvider
			}
		}
	case TransportHTTP:
		if os.Getenv(envMesheryToken) != "" {
			return nil, ErrTokenNotAllowedForHTTP()
		}
		cfg.Transport.Stdio = StdioConfig{}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// host is already lowercase.
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
		// Local ignores tokens: requests would silently run as the local user.
		if c.Transport.Stdio.Token != "" && isLocalProvider(c.Transport.Stdio.Provider) {
			return ErrTokenWithLocalProvider(c.Transport.Stdio.Provider)
		}
	case TransportHTTP:
		if err := c.Transport.HTTP.validate(); err != nil {
			return err
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

func (h *HTTPConfig) validate() error {
	if !isLoopback(h.Host) {
		return ErrNonLoopbackHost(h.Host)
	}
	if h.Port < 1 || h.Port > 65535 {
		return ErrInvalidPort(h.Port)
	}
	return nil
}
