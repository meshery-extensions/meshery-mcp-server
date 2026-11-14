package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meshery/meshkit/errors"
)

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad(t *testing.T) {
	const fileYAML = "meshery:\n  url: https://file.example:9081\n  token: file-token\nlog:\n  level: warn\n"

	tests := []struct {
		name              string
		envURL            string
		envToken          string
		envLevel          string
		envKeychainScope  string
		envMaxConcurrent  string
		envMaxRetries     string
		fileYAML          string // "" = no config file
		path              string // explicit path; overrides fileYAML
		wantURL           string
		wantToken         Secret
		wantLevel         string
		wantKeychainScope string
		wantMaxConcurrent int
		wantMaxRetries    int
		wantErr           string // expected short description; "" = expect success
	}{
		{name: "defaults only", wantURL: defaultMesheryURL, wantLevel: LogLevelInfo,
			wantMaxConcurrent: defaultMaxConcurrentCalls, wantMaxRetries: defaultMaxRetries},
		{name: "env overrides default", envURL: "https://env.example", envToken: "env-token", envLevel: "debug",
			wantURL: "https://env.example", wantToken: "env-token", wantLevel: LogLevelDebug,
			wantMaxConcurrent: defaultMaxConcurrentCalls, wantMaxRetries: defaultMaxRetries},
		{name: "file only", fileYAML: fileYAML,
			wantURL: "https://file.example:9081", wantToken: "file-token", wantLevel: LogLevelWarn,
			wantMaxConcurrent: defaultMaxConcurrentCalls, wantMaxRetries: defaultMaxRetries},
		{name: "env beats file per key", fileYAML: fileYAML, envURL: "https://env.example",
			wantURL: "https://env.example", wantToken: "file-token", wantLevel: LogLevelWarn,
			wantMaxConcurrent: defaultMaxConcurrentCalls, wantMaxRetries: defaultMaxRetries},
		{name: "log level is case-insensitive", envLevel: " DEBUG ",
			wantURL: defaultMesheryURL, wantLevel: LogLevelDebug,
			wantMaxConcurrent: defaultMaxConcurrentCalls, wantMaxRetries: defaultMaxRetries},
		{name: "keychain scope from env", envKeychainScope: "org:admin",
			wantURL: defaultMesheryURL, wantLevel: LogLevelInfo, wantKeychainScope: "org:admin",
			wantMaxConcurrent: defaultMaxConcurrentCalls, wantMaxRetries: defaultMaxRetries},
		{name: "max concurrent calls from env", envMaxConcurrent: "8",
			wantURL: defaultMesheryURL, wantLevel: LogLevelInfo, wantMaxConcurrent: 8, wantMaxRetries: defaultMaxRetries},
		{name: "max retries from env", envMaxRetries: "7",
			wantURL: defaultMesheryURL, wantLevel: LogLevelInfo, wantMaxRetries: 7, wantMaxConcurrent: defaultMaxConcurrentCalls},
		{name: "repo config_example.yml loads", path: filepath.Join("..", "..", "config_example.yml"),
			wantURL: defaultMesheryURL, wantLevel: LogLevelInfo,
			wantMaxConcurrent: defaultMaxConcurrentCalls, wantMaxRetries: defaultMaxRetries},
		{name: "invalid scheme", envURL: "ftp://x", wantErr: "Invalid Meshery server URL"},
		{name: "missing scheme", envURL: "localhost:9081", wantErr: "Invalid Meshery server URL"},
		{name: "unparsable URL", envURL: "http://u:p@host:90x81", wantErr: "Invalid Meshery server URL"},
		{name: "invalid log level", envLevel: "verbose", wantErr: "Invalid log level"},
		{name: "max concurrent calls below 1", envMaxConcurrent: "0", wantErr: "Invalid max concurrent calls"},
		{name: "max retries negative", envMaxRetries: "-1", wantErr: "Invalid max retries"},
		{name: "missing file", path: filepath.Join(t.TempDir(), "nope.yaml"), wantErr: "Unable to read the config file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Always set all: an empty value counts as unset.
			t.Setenv(envMesheryURL, tt.envURL)
			t.Setenv(envMesheryToken, tt.envToken)
			t.Setenv(envMesheryKeychainScope, tt.envKeychainScope)
			t.Setenv(envMesheryMaxConcurrentCalls, tt.envMaxConcurrent)
			t.Setenv(envMesheryMaxRetries, tt.envMaxRetries)
			t.Setenv(envLogLevel, tt.envLevel)

			path := tt.path
			if path == "" && tt.fileYAML != "" {
				path = writeConfigFile(t, tt.fileYAML)
			}

			cfg, err := Load(path)
			if tt.wantErr != "" {
				if err == nil || errors.GetSDescription(err) != tt.wantErr {
					t.Fatalf("want error %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Meshery.URL != tt.wantURL {
				t.Errorf("URL = %q, want %q", cfg.Meshery.URL, tt.wantURL)
			}
			if cfg.Meshery.Token != tt.wantToken {
				t.Errorf("Token = %q, want %q", string(cfg.Meshery.Token), string(tt.wantToken))
			}
			if cfg.Log.Level != tt.wantLevel {
				t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, tt.wantLevel)
			}
			if cfg.Meshery.KeychainScope != tt.wantKeychainScope {
				t.Errorf("KeychainScope = %q, want %q", cfg.Meshery.KeychainScope, tt.wantKeychainScope)
			}
			if cfg.Meshery.MaxConcurrentCalls != tt.wantMaxConcurrent {
				t.Errorf("MaxConcurrentCalls = %d, want %d", cfg.Meshery.MaxConcurrentCalls, tt.wantMaxConcurrent)
			}
			if cfg.Meshery.MaxRetries != tt.wantMaxRetries {
				t.Errorf("MaxRetries = %d, want %d", cfg.Meshery.MaxRetries, tt.wantMaxRetries)
			}
		})
	}
}

func TestInvalidURLErrorDoesNotLeakCredentials(t *testing.T) {
	t.Setenv(envMesheryToken, "")
	t.Setenv(envLogLevel, "")
	for _, raw := range []string{"http://admin:s3cret@host:90x81", "ftp://admin:s3cret@host"} {
		t.Setenv(envMesheryURL, raw)
		_, err := Load("")
		if err == nil {
			t.Fatalf("%s: expected an error", raw)
		}
		if strings.Contains(err.Error(), "s3cret") {
			t.Errorf("password leaked into error: %v", err)
		}
	}
}

func TestSecretIsRedacted(t *testing.T) {
	cfg := Config{Meshery: MesheryConfig{URL: "http://x", Token: "super-secret"}}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, out := range []string{fmt.Sprint(cfg), fmt.Sprintf("%+v", cfg), string(b)} {
		if strings.Contains(out, "super-secret") {
			t.Errorf("token leaked: %s", out)
		}
	}
}
