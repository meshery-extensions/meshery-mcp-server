// Package config loads and validates meshery-mcp-server configuration.
//
// Precedence, lowest to highest: defaults < config file < environment variables.
package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/viper"
)

const (
	keyMesheryURL   = "meshery.url"
	keyMesheryToken = "meshery.token"

	keyMesheryKeychainScope      = "meshery.keychain_scope"
	keyMesheryMaxConcurrentCalls = "meshery.max_concurrent_calls"
	keyMesheryMaxRetries         = "meshery.max_retries"
	keyLogLevel                  = "log.level"

	envMesheryURL   = "MESHERY_SERVER_URL"
	envMesheryToken = "MESHERY_API_TOKEN" //nolint:gosec // G101: env var name, not a credential

	envMesheryKeychainScope      = "MESHERY_KEYCHAIN_SCOPE"
	envMesheryMaxConcurrentCalls = "MESHERY_MAX_CONCURRENT_CALLS"
	envMesheryMaxRetries         = "MESHERY_MAX_RETRIES"
	envLogLevel                  = "LOG_LEVEL"

	defaultMesheryURL         = "http://localhost:9081"
	defaultMaxConcurrentCalls = 4
	defaultMaxRetries         = 3
	defaultLogLevel           = LogLevelInfo
)

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

// Config is the fully resolved, validated configuration.
type Config struct {
	Meshery MesheryConfig `mapstructure:"meshery"`
	Log     LogConfig     `mapstructure:"log"`
}

// MesheryConfig describes how to reach the Meshery server and how to
// authenticate against Meshery Cloud.
type MesheryConfig struct {
	URL   string `mapstructure:"url"`
	Token Secret `mapstructure:"token"`

	// KeychainScope is the narrowest keychain scope the deployment needs.
	// Empty (the default) requests no keychain scope: unscoped tokens
	// behave as before.
	KeychainScope string `mapstructure:"keychain_scope"`

	// MaxConcurrentCalls bounds concurrent Cloud calls per tool.
	MaxConcurrentCalls int `mapstructure:"max_concurrent_calls"`

	// MaxRetries bounds 429/503 retries per Cloud call.
	MaxRetries int `mapstructure:"max_retries"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level string `mapstructure:"level"` // one of the LogLevel* constants
}

// Load builds the configuration. configPath is optional; an empty string
// means no file is read.
func Load(configPath string) (*Config, error) {
	v := viper.New()
	v.SetDefault(keyMesheryURL, defaultMesheryURL)
	v.SetDefault(keyMesheryToken, "")
	v.SetDefault(keyMesheryKeychainScope, "")
	v.SetDefault(keyMesheryMaxConcurrentCalls, defaultMaxConcurrentCalls)
	v.SetDefault(keyMesheryMaxRetries, defaultMaxRetries)
	v.SetDefault(keyLogLevel, defaultLogLevel)

	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, ErrReadConfigFile(err, configPath)
		}
	}

	// MustBindEnv only fails when called without a key, which would be a bug
	// here, not something a user can fix; panicking is appropriate.
	v.MustBindEnv(keyMesheryURL, envMesheryURL)
	v.MustBindEnv(keyMesheryToken, envMesheryToken)
	v.MustBindEnv(keyMesheryKeychainScope, envMesheryKeychainScope)
	v.MustBindEnv(keyMesheryMaxConcurrentCalls, envMesheryMaxConcurrentCalls)
	v.MustBindEnv(keyMesheryMaxRetries, envMesheryMaxRetries)
	v.MustBindEnv(keyLogLevel, envLogLevel)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, ErrDecodeConfig(err)
	}
	cfg.Log.Level = strings.ToLower(strings.TrimSpace(cfg.Log.Level))

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
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

	switch c.Log.Level {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
	default:
		return ErrInvalidLogLevel(c.Log.Level)
	}

	if c.Meshery.MaxConcurrentCalls < 1 {
		return ErrInvalidMaxConcurrentCalls(c.Meshery.MaxConcurrentCalls)
	}
	if c.Meshery.MaxRetries < 0 {
		return ErrInvalidMaxRetries(c.Meshery.MaxRetries)
	}
	return nil
}
