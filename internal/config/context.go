// Copyright Meshery Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultConfigFileName is the default configuration file name.
	DefaultConfigFileName = "mcp-config.yaml"
	// DefaultConfigDir is the default directory for Meshery configuration.
	DefaultConfigDir = ".meshery"
)

var (
	// ErrContextNotFound is returned when a context is not found.
	ErrContextNotFound = errors.New("context not found")
	// ErrNoContexts is returned when no contexts are configured.
	ErrNoContexts = errors.New("no contexts configured")
	// ErrConfigNotLoaded is returned when config file was not loaded.
	ErrConfigNotLoaded = errors.New("config file not loaded; using environment variables")
)

// Context represents a single Meshery instance configuration.
type Context struct {
	// Server is the base URL of the Meshery Server REST API.
	Server string `yaml:"server"`
	// Token is the API token for authentication.
	Token string `yaml:"token"`
}

// MultiContextConfig holds the full configuration with multiple contexts.
type MultiContextConfig struct {
	// CurrentContext is the name of the currently active context.
	CurrentContext string `yaml:"current-context"`
	// Contexts is a map of context names to their configurations.
	Contexts map[string]Context `yaml:"contexts"`

	mu         sync.RWMutex
	configPath string
	fromFile   bool
}

// Manager manages the multi-context configuration.
type Manager struct {
	config *MultiContextConfig
	mu     sync.RWMutex
}

// NewManager creates a new configuration manager.
func NewManager() *Manager {
	return &Manager{}
}

// Load loads configuration with the following priority:
// 1. Environment variables (MESHERY_SERVER_URL, MESHERY_API_TOKEN)
// 2. Config file (~/.meshery/mcp-config.yaml or MESHERY_CONFIG_PATH)
//
// Environment variables always override config file values for the active context.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try to load from config file first
	configPath := m.getConfigPath()
	cfg, err := m.loadFromFile(configPath)
	if err != nil {
		// If file doesn't exist, create a default config from env vars
		if os.IsNotExist(err) {
			cfg = m.createFromEnv()
		} else {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Apply environment variable overrides
	m.applyEnvOverrides(cfg)

	m.config = cfg
	return nil
}

// getConfigPath returns the configuration file path.
func (m *Manager) getConfigPath() string {
	if path := os.Getenv("MESHERY_CONFIG_PATH"); path != "" {
		return path
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, DefaultConfigDir, DefaultConfigFileName)
}

// loadFromFile loads configuration from a YAML file.
func (m *Manager) loadFromFile(path string) (*MultiContextConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg MultiContextConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg.configPath = path
	cfg.fromFile = true
	return &cfg, nil
}

// createFromEnv creates a configuration from environment variables.
func (m *Manager) createFromEnv() *MultiContextConfig {
	serverURL := envOr("MESHERY_SERVER_URL", DefaultMeshServerURL)
	token := os.Getenv("MESHERY_API_TOKEN")

	return &MultiContextConfig{
		CurrentContext: "default",
		Contexts: map[string]Context{
			"default": {
				Server: serverURL,
				Token:  token,
			},
		},
		fromFile: false,
	}
}

// applyEnvOverrides applies environment variable overrides to the active context.
func (m *Manager) applyEnvOverrides(cfg *MultiContextConfig) {
	// MESHERY_CONTEXT overrides the current context selection
	if ctxName := os.Getenv("MESHERY_CONTEXT"); ctxName != "" {
		if _, exists := cfg.Contexts[ctxName]; exists {
			cfg.CurrentContext = ctxName
		}
	}

	// Get the current context
	ctx, exists := cfg.Contexts[cfg.CurrentContext]
	if !exists {
		return
	}

	// Environment variables override config file values
	if serverURL := os.Getenv("MESHERY_SERVER_URL"); serverURL != "" {
		ctx.Server = serverURL
	}
	if token := os.Getenv("MESHERY_API_TOKEN"); token != "" {
		ctx.Token = token
	}

	cfg.Contexts[cfg.CurrentContext] = ctx
}

// GetCurrentContext returns the currently active context.
func (m *Manager) GetCurrentContext() (*Context, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return nil, "", ErrConfigNotLoaded
	}

	ctx, exists := m.config.Contexts[m.config.CurrentContext]
	if !exists {
		return nil, "", ErrContextNotFound
	}

	return &ctx, m.config.CurrentContext, nil
}

// GetContext returns a specific context by name.
func (m *Manager) GetContext(name string) (*Context, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return nil, ErrConfigNotLoaded
	}

	ctx, exists := m.config.Contexts[name]
	if !exists {
		return nil, ErrContextNotFound
	}

	return &ctx, nil
}

// ListContexts returns all configured contexts.
func (m *Manager) ListContexts() (map[string]Context, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return nil, "", ErrConfigNotLoaded
	}

	if len(m.config.Contexts) == 0 {
		return nil, "", ErrNoContexts
	}

	// Return a copy to prevent external modification
	contexts := make(map[string]Context, len(m.config.Contexts))
	for k, v := range m.config.Contexts {
		contexts[k] = v
	}

	return contexts, m.config.CurrentContext, nil
}

// SwitchContext switches the active context to the specified name.
func (m *Manager) SwitchContext(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.config == nil {
		return ErrConfigNotLoaded
	}

	if _, exists := m.config.Contexts[name]; !exists {
		return fmt.Errorf("%w: %s", ErrContextNotFound, name)
	}

	m.config.CurrentContext = name
	return nil
}

// IsFromFile returns true if the configuration was loaded from a file.
func (m *Manager) IsFromFile() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return false
	}
	return m.config.fromFile
}

// GetConfig returns a Config compatible with the existing interface.
func (m *Manager) GetConfig() *Config {
	ctx, _, err := m.GetCurrentContext()
	if err != nil {
		return Load() // Fall back to simple env-based loading
	}

	return &Config{
		MeshServerURL: ctx.Server,
		MeshAPIToken:  ctx.Token,
	}
}

// SetupPrompt returns a helpful message when no config file is found.
func SetupPrompt() string {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, DefaultConfigDir, DefaultConfigFileName)

	return fmt.Sprintf(`No configuration file found. You can:

1. Create a config file at %s:

   current-context: dev
   contexts:
     dev:
       server: http://localhost:9081
       token: <your-meshery-api-token>
     production:
       server: https://meshery.example.com
       token: <your-meshery-api-token>

2. Or use environment variables for a quick setup:

   export MESHERY_SERVER_URL=http://localhost:9081
   export MESHERY_API_TOKEN=<your-meshery-api-token>

Currently using defaults: server=%s, token=<not set>
`, configPath, DefaultMeshServerURL)
}
