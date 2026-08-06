package meshery

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Config holds configuration settings for the internal Meshery REST client.
type Config struct {
	BaseURL    string        `json:"base_url" yaml:"base_url"`
	Token      string        `json:"token" yaml:"token"`
	Timeout    time.Duration `json:"timeout" yaml:"timeout"`
	RetryCount int           `json:"retry_count" yaml:"retry_count"`
	UserAgent  string        `json:"user_agent" yaml:"user_agent"`
	// HTTPClient allows supplying a custom http.Client. When supplied, its Timeout field
	// will be set to Config.Timeout.
	HTTPClient *http.Client `json:"-" yaml:"-"`
}

// DefaultConfig returns a Config initialized with sensible default values.
func DefaultConfig() Config {
	return Config{
		BaseURL:    "http://localhost:9081",
		Timeout:    10 * time.Second,
		RetryCount: 2,
		UserAgent:  "meshery-mcp-server/0.1.0",
	}
}

// LoadConfig constructs a Config by applying setting sources in explicit precedence order:
// Environment Variables (MESHERY_SERVER_URL, MESHERY_API_TOKEN) > Config File (JSON) > Defaults.
func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return cfg, fmt.Errorf("failed to read config file %q: %w", path, err)
		}

		var fileCfg Config
		if err := json.Unmarshal(data, &fileCfg); err != nil {
			return cfg, fmt.Errorf("failed to parse config file %q: %w", path, err)
		}

		if fileCfg.BaseURL != "" {
			cfg.BaseURL = fileCfg.BaseURL
		}
		if fileCfg.Token != "" {
			cfg.Token = fileCfg.Token
		}
		if fileCfg.Timeout > 0 {
			cfg.Timeout = fileCfg.Timeout
		}
		if fileCfg.RetryCount > 0 {
			cfg.RetryCount = fileCfg.RetryCount
		}
		if fileCfg.UserAgent != "" {
			cfg.UserAgent = fileCfg.UserAgent
		}
	}

	if envURL := os.Getenv("MESHERY_SERVER_URL"); envURL != "" {
		cfg.BaseURL = envURL
	}
	if envToken := os.Getenv("MESHERY_API_TOKEN"); envToken != "" {
		cfg.Token = envToken
	}

	return cfg, nil
}
