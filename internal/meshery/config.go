package meshery

import (
	"net/http"
	"time"
)

// Config holds configuration settings for the internal Meshery REST client.
type Config struct {
	BaseURL    string        `json:"base_url" yaml:"base_url"`
	Token      string        `json:"token" yaml:"token"`
	Timeout    time.Duration `json:"timeout" yaml:"timeout"`
	RetryCount int           `json:"retry_count" yaml:"retry_count"`
	UserAgent  string        `json:"user_agent" yaml:"user_agent"`
	HTTPClient *http.Client  `json:"-" yaml:"-"`
}

// DefaultConfig returns a Config initialized with sensible default values.
func DefaultConfig() Config {
	return Config{
		BaseURL:    "http://localhost:9081",
		Timeout:    10 * time.Second,
		RetryCount: 2,
		UserAgent:  "meshery-mcp-server",
	}
}
