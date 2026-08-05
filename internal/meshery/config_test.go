package meshery

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.BaseURL != "http://localhost:9081" {
		t.Errorf("expected default BaseURL 'http://localhost:9081', got %q", cfg.BaseURL)
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("expected default Timeout 10s, got %v", cfg.Timeout)
	}
	if cfg.RetryCount != 2 {
		t.Errorf("expected default RetryCount 2, got %d", cfg.RetryCount)
	}
	if cfg.UserAgent != "meshery-mcp-server/0.1.0" {
		t.Errorf("expected default UserAgent 'meshery-mcp-server/0.1.0', got %q", cfg.UserAgent)
	}
}

func TestLoadConfig_ConfigFile_PartialOverride(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "config.json")
	content := `{
		"base_url": "https://meshery.example.com",
		"token": "file-token-123"
	}`

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp config file: %v", err)
	}

	cfg, err := LoadConfig(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.BaseURL != "https://meshery.example.com" {
		t.Errorf("expected BaseURL from file 'https://meshery.example.com', got %q", cfg.BaseURL)
	}
	if cfg.Token != "file-token-123" {
		t.Errorf("expected Token from file 'file-token-123', got %q", cfg.Token)
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("expected default Timeout 10s preserved, got %v", cfg.Timeout)
	}
	if cfg.RetryCount != 2 {
		t.Errorf("expected default RetryCount 2 preserved, got %d", cfg.RetryCount)
	}
}

func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "config.json")
	content := `{
		"base_url": "https://file.example.com",
		"token": "file-token"
	}`

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp config file: %v", err)
	}

	t.Setenv("MESHERY_SERVER_URL", "https://env.example.com:9081")
	t.Setenv("MESHERY_API_TOKEN", "env-token-xyz")

	cfg, err := LoadConfig(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.BaseURL != "https://env.example.com:9081" {
		t.Errorf("expected BaseURL from env 'https://env.example.com:9081', got %q", cfg.BaseURL)
	}
	if cfg.Token != "env-token-xyz" {
		t.Errorf("expected Token from env 'env-token-xyz', got %q", cfg.Token)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	t.Parallel()

	_, err := LoadConfig("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(filePath, []byte(`{invalid-json`), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err := LoadConfig(filePath)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
