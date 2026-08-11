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
	"os"
	"path/filepath"
	"testing"
)

func TestManagerLoadFromEnv(t *testing.T) {
	// Clear any existing config
	t.Setenv("MESHERY_CONFIG_PATH", "/nonexistent/path/config.yaml")
	t.Setenv("MESHERY_SERVER_URL", "http://test-server:8080")
	t.Setenv("MESHERY_API_TOKEN", "test-token-123")

	mgr := NewManager()
	if err := mgr.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	ctx, name, err := mgr.GetCurrentContext()
	if err != nil {
		t.Fatalf("GetCurrentContext() error = %v", err)
	}

	if name != "default" {
		t.Errorf("context name = %q, want %q", name, "default")
	}

	if ctx.Server != "http://test-server:8080" {
		t.Errorf("Server = %q, want %q", ctx.Server, "http://test-server:8080")
	}

	if ctx.Token != "test-token-123" {
		t.Errorf("Token = %q, want %q", ctx.Token, "test-token-123")
	}

	if mgr.IsFromFile() {
		t.Error("IsFromFile() should be false when loaded from env vars")
	}
}

func TestManagerLoadFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcp-config.yaml")

	configContent := `current-context: staging
contexts:
  dev:
    server: http://localhost:9081
    token: dev-token
  staging:
    server: https://meshery.staging.example.com
    token: staging-token
  production:
    server: https://meshery.example.com
    token: prod-token
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	t.Setenv("MESHERY_CONFIG_PATH", configPath)
	t.Setenv("MESHERY_SERVER_URL", "")
	t.Setenv("MESHERY_API_TOKEN", "")

	mgr := NewManager()
	if err := mgr.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	ctx, name, err := mgr.GetCurrentContext()
	if err != nil {
		t.Fatalf("GetCurrentContext() error = %v", err)
	}

	if name != "staging" {
		t.Errorf("context name = %q, want %q", name, "staging")
	}

	if ctx.Server != "https://meshery.staging.example.com" {
		t.Errorf("Server = %q, want %q", ctx.Server, "https://meshery.staging.example.com")
	}

	if !mgr.IsFromFile() {
		t.Error("IsFromFile() should be true when loaded from file")
	}
}

func TestManagerSwitchContext(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcp-config.yaml")

	configContent := `current-context: dev
contexts:
  dev:
    server: http://localhost:9081
    token: dev-token
  production:
    server: https://meshery.example.com
    token: prod-token
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	t.Setenv("MESHERY_CONFIG_PATH", configPath)
	t.Setenv("MESHERY_SERVER_URL", "")
	t.Setenv("MESHERY_API_TOKEN", "")

	mgr := NewManager()
	if err := mgr.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify initial context
	_, name, _ := mgr.GetCurrentContext()
	if name != "dev" {
		t.Errorf("initial context = %q, want %q", name, "dev")
	}

	// Switch to production
	if err := mgr.SwitchContext("production"); err != nil {
		t.Fatalf("SwitchContext() error = %v", err)
	}

	ctx, name, _ := mgr.GetCurrentContext()
	if name != "production" {
		t.Errorf("switched context name = %q, want %q", name, "production")
	}
	if ctx.Server != "https://meshery.example.com" {
		t.Errorf("switched Server = %q, want %q", ctx.Server, "https://meshery.example.com")
	}

	// Try switching to non-existent context
	err := mgr.SwitchContext("nonexistent")
	if err == nil {
		t.Error("SwitchContext() should error for non-existent context")
	}
}

func TestManagerListContexts(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcp-config.yaml")

	configContent := `current-context: dev
contexts:
  dev:
    server: http://localhost:9081
    token: dev-token
  staging:
    server: https://meshery.staging.example.com
    token: staging-token
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	t.Setenv("MESHERY_CONFIG_PATH", configPath)
	t.Setenv("MESHERY_SERVER_URL", "")
	t.Setenv("MESHERY_API_TOKEN", "")

	mgr := NewManager()
	if err := mgr.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	contexts, current, err := mgr.ListContexts()
	if err != nil {
		t.Fatalf("ListContexts() error = %v", err)
	}

	if current != "dev" {
		t.Errorf("current context = %q, want %q", current, "dev")
	}

	if len(contexts) != 2 {
		t.Errorf("len(contexts) = %d, want %d", len(contexts), 2)
	}

	if _, ok := contexts["dev"]; !ok {
		t.Error("contexts should contain 'dev'")
	}

	if _, ok := contexts["staging"]; !ok {
		t.Error("contexts should contain 'staging'")
	}
}

func TestManagerEnvOverrides(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcp-config.yaml")

	configContent := `current-context: dev
contexts:
  dev:
    server: http://localhost:9081
    token: dev-token
  staging:
    server: https://meshery.staging.example.com
    token: staging-token
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Set env vars to override
	t.Setenv("MESHERY_CONFIG_PATH", configPath)
	t.Setenv("MESHERY_CONTEXT", "staging")
	t.Setenv("MESHERY_SERVER_URL", "http://override-server:9999")
	t.Setenv("MESHERY_API_TOKEN", "override-token")

	mgr := NewManager()
	if err := mgr.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	ctx, name, _ := mgr.GetCurrentContext()

	// MESHERY_CONTEXT should override current-context
	if name != "staging" {
		t.Errorf("context name = %q, want %q (overridden by env)", name, "staging")
	}

	// MESHERY_SERVER_URL should override the context's server
	if ctx.Server != "http://override-server:9999" {
		t.Errorf("Server = %q, want %q (overridden by env)", ctx.Server, "http://override-server:9999")
	}

	// MESHERY_API_TOKEN should override the context's token
	if ctx.Token != "override-token" {
		t.Errorf("Token = %q, want %q (overridden by env)", ctx.Token, "override-token")
	}
}

func TestSetupPrompt(t *testing.T) {
	prompt := SetupPrompt()

	if prompt == "" {
		t.Error("SetupPrompt() should return non-empty string")
	}

	// Check that it contains expected content
	if !contains(prompt, "mcp-config.yaml") {
		t.Error("SetupPrompt should mention config file")
	}

	if !contains(prompt, "MESHERY_SERVER_URL") {
		t.Error("SetupPrompt should mention MESHERY_SERVER_URL env var")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
