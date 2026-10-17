package config

import (
	"encoding/json"
	"fmt"
	"net/url"
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
	const fileYAML = "meshery:\n  url: https://file.example:9081\nlog:\n  level: warn\n"

	tests := []struct {
		name      string
		envURL    string
		envToken  string
		envLevel  string
		fileYAML  string // "" = no config file
		path      string // explicit path; overrides fileYAML
		wantURL   string
		wantToken Secret
		wantLevel string
		wantErr   string // expected short description; "" = expect success
	}{
		{name: "defaults only", wantURL: defaultMesheryURL, wantLevel: LogLevelInfo},
		{name: "env overrides default", envURL: "https://env.example", envToken: "env-token", envLevel: "debug",
			wantURL: "https://env.example", wantToken: "env-token", wantLevel: LogLevelDebug},
		{name: "file only", fileYAML: fileYAML,
			wantURL: "https://file.example:9081", wantLevel: LogLevelWarn},
		{name: "env beats file per key", fileYAML: fileYAML, envURL: "https://env.example", envToken: "env-token",
			wantURL: "https://env.example", wantToken: "env-token", wantLevel: LogLevelWarn},
		{name: "log level is case-insensitive", envLevel: " DEBUG ",
			wantURL: defaultMesheryURL, wantLevel: LogLevelDebug},
		{name: "repo config_example.yml loads", path: filepath.Join("..", "..", "config_example.yml"),
			wantURL: defaultMesheryURL, wantLevel: LogLevelInfo},
		{name: "invalid scheme", envURL: "ftp://x", wantErr: "Invalid Meshery server URL"},
		{name: "missing scheme", envURL: "localhost:9081", wantErr: "Invalid Meshery server URL"},
		{name: "unparsable URL", envURL: "http://u:p@host:90x81", wantErr: "Invalid Meshery server URL"},
		{name: "invalid log level", envLevel: "verbose", wantErr: "Invalid log level"},
		{name: "missing file", path: filepath.Join(t.TempDir(), "nope.yaml"), wantErr: "Unable to read the config file"},

		// Token
		{name: "token from the environment", envToken: "env-token",
			wantURL: defaultMesheryURL, wantToken: "env-token", wantLevel: LogLevelInfo},
		{name: "token under meshery in file", fileYAML: "meshery:\n  token: file-token\n",
			wantErr: "Token is not allowed in the config file"},
		{name: "token under auth in file", fileYAML: "auth:\n  token: file-token\n",
			wantErr: "Token is not allowed in the config file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Always set all: an empty value counts as unset.
			t.Setenv(envMesheryURL, tt.envURL)
			t.Setenv(envMesheryToken, tt.envToken)
			t.Setenv(envLogLevel, tt.envLevel)
			t.Setenv(envLogFormat, "")
			clearTransportEnv(t)

			path := tt.path
			if path == "" && tt.fileYAML != "" {
				path = writeConfigFile(t, tt.fileYAML)
			}

			cfg, err := Load(path, Flags{})
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
			if cfg.Auth.Token != tt.wantToken {
				t.Errorf("Auth.Token = %q, want %q", string(cfg.Auth.Token), string(tt.wantToken))
			}
			if cfg.Log.Level != tt.wantLevel {
				t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, tt.wantLevel)
			}
		})
	}
}

func clearTransportEnv(t *testing.T) {
	t.Helper()
	t.Setenv(envTransportMode, "")
	t.Setenv(envTransportHost, "")
	t.Setenv(envTransportPort, "")
	t.Setenv(envAccessKey, "")
}

func TestLoad_Transport(t *testing.T) {
	str := func(s string) *string { return &s }
	num := func(n int) *int { return &n }
	key := Secret(strings.Repeat("k", MinAccessKeyLength))

	tests := []struct {
		name     string
		envMode  string
		envHost  string
		envPort  string
		envKey   Secret
		fileYAML string
		flags    Flags
		want     TransportConfig
		wantErr  string
	}{
		{name: "defaults to stdio",
			want: TransportConfig{Mode: TransportStdio, Host: defaultTransportHost, Port: defaultTransportPort}},
		{name: "http from env", envMode: "HTTP", envPort: "9000", envKey: key,
			want: TransportConfig{Mode: TransportHTTP, Host: defaultTransportHost, Port: 9000, AccessKey: key}},
		{name: "http from file", fileYAML: "transport:\n  mode: http\n  host: localhost\n  port: 7000\n", envKey: key,
			want: TransportConfig{Mode: TransportHTTP, Host: "localhost", Port: 7000, AccessKey: key}},
		{name: "flags beat env", envMode: "stdio", envPort: "9000", envKey: key,
			flags: Flags{Transport: str("http"), Port: num(8181)},
			want:  TransportConfig{Mode: TransportHTTP, Host: defaultTransportHost, Port: 8181, AccessKey: key}},
		{name: "IPv6 loopback", flags: Flags{Transport: str("http"), Host: str("::1")}, envKey: key,
			want: TransportConfig{Mode: TransportHTTP, Host: "::1", Port: defaultTransportPort, AccessKey: key}},
		{name: "host is lowercased", flags: Flags{Transport: str("http"), Host: str("LocalHost")}, envKey: key,
			want: TransportConfig{Mode: TransportHTTP, Host: "localhost", Port: defaultTransportPort, AccessKey: key}},
		{name: "stdio ignores host", flags: Flags{Host: str("0.0.0.0")},
			want: TransportConfig{Mode: TransportStdio, Host: "0.0.0.0", Port: defaultTransportPort}},
		{name: "stdio does not read the access key", envKey: key,
			want: TransportConfig{Mode: TransportStdio, Host: defaultTransportHost, Port: defaultTransportPort}},
		{name: "http without access key", flags: Flags{Transport: str("http")},
			wantErr: "HTTP access key is required"},
		{name: "http with short access key", flags: Flags{Transport: str("http")}, envKey: "too-short",
			wantErr: "HTTP access key is required"},
		{name: "access key in file", fileYAML: "transport:\n  access_key: in-file\n",
			wantErr: "Access key is not allowed in the config file"},
		{name: "camelCase access key in file", fileYAML: "transport:\n  accessKey: in-file\n",
			wantErr: "Access key is not allowed in the config file"},
		{name: "non-loopback host", flags: Flags{Transport: str("http"), Host: str("0.0.0.0")}, envKey: key,
			wantErr: "HTTP transport must listen on a loopback address"},
		{name: "hostname that is not localhost", flags: Flags{Transport: str("http"), Host: str("example.com")}, envKey: key,
			wantErr: "HTTP transport must listen on a loopback address"},
		{name: "port out of range", flags: Flags{Transport: str("http"), Port: num(70000)}, envKey: key,
			wantErr: "Invalid port"},
		{name: "port not a number", envMode: "http", envPort: "eighty", envKey: key,
			wantErr: "Unable to decode configuration"},
		{name: "sse is not supported", flags: Flags{Transport: str("sse")},
			wantErr: "Invalid transport"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(envMesheryURL, "")
			t.Setenv(envMesheryToken, "")
			t.Setenv(envLogLevel, "")
			t.Setenv(envLogFormat, "")
			t.Setenv(envTransportMode, tt.envMode)
			t.Setenv(envTransportHost, tt.envHost)
			t.Setenv(envTransportPort, tt.envPort)
			t.Setenv(envAccessKey, string(tt.envKey))

			path := ""
			if tt.fileYAML != "" {
				path = writeConfigFile(t, tt.fileYAML)
			}

			cfg, err := Load(path, tt.flags)
			if tt.wantErr != "" {
				if err == nil || errors.GetSDescription(err) != tt.wantErr {
					t.Fatalf("want error %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Transport != tt.want {
				t.Errorf("Transport = %+v, want %+v", cfg.Transport, tt.want)
			}
		})
	}
}

func TestAccessKeyErrorsDoNotLeakKey(t *testing.T) {
	clearTransportEnv(t)
	t.Setenv(envMesheryURL, "")
	t.Setenv(envMesheryToken, "")
	t.Setenv(envLogLevel, "")
	t.Setenv(envLogFormat, "")
	http := "http"

	t.Run("too short", func(t *testing.T) {
		t.Setenv(envAccessKey, "short-secret")
		_, err := Load("", Flags{Transport: &http})
		if err == nil || strings.Contains(err.Error(), "short-secret") {
			t.Errorf("want an error without the key, got %v", err)
		}
	})
	t.Run("in config file", func(t *testing.T) {
		_, err := Load(writeConfigFile(t, "transport:\n  access_key: file-secret\n"), Flags{})
		if err == nil || strings.Contains(err.Error(), "file-secret") {
			t.Errorf("want an error without the key, got %v", err)
		}
	})
}

func TestInvalidURLErrorDoesNotLeakCredentials(t *testing.T) {
	t.Setenv(envMesheryToken, "")
	t.Setenv(envLogLevel, "")
	t.Setenv(envLogFormat, "")
	clearTransportEnv(t)
	for _, raw := range []string{"http://admin:s3cret@host:90x81", "ftp://admin:s3cret@host"} {
		t.Setenv(envMesheryURL, raw)
		_, err := Load("", Flags{})
		if err == nil {
			t.Fatalf("%s: expected an error", raw)
		}
		if strings.Contains(err.Error(), "s3cret") {
			t.Errorf("password leaked into error: %v", err)
		}
	}
}

func TestTokenErrorsDoNotLeakToken(t *testing.T) {
	t.Setenv(envMesheryURL, "")
	t.Setenv(envLogLevel, "")
	t.Setenv(envLogFormat, "")
	t.Setenv(envMesheryToken, "super-secret")
	clearTransportEnv(t)

	_, err := Load(writeConfigFile(t, "meshery:\n  token: file-secret\n"), Flags{})
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, secret := range []string{"super-secret", "file-secret"} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("token leaked into error: %v", err)
		}
	}
}

// urlWithPassword builds a URL that embeds a password. It is built at run
// time so the test holds no URL literal with credentials, which gosec
// reports as hardcoded (G101).
func urlWithPassword(password string) string {
	u := url.URL{Scheme: "https", User: url.UserPassword("admin", password), Host: "meshery.example"}
	return u.String()
}

func TestRedactedURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{url: "http://localhost:9081", want: "http://localhost:9081"},
		{url: urlWithPassword("s3cret"), want: urlWithPassword("xxxxx")},
		{url: "http://u:p@host:90x81", want: "<unparsable URL>"},
	}
	for _, tt := range tests {
		got := MesheryConfig{URL: tt.url}.RedactedURL()
		if got != tt.want {
			t.Errorf("RedactedURL(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestSecretIsRedacted(t *testing.T) {
	cfg := Config{
		Meshery: MesheryConfig{URL: "http://x"},
		Auth:    AuthConfig{Token: "super-secret"},
	}
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

func TestLoad_LogFormat(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		fileYAML string
		want     string
		wantErr  string
	}{
		{name: "defaults to text", want: LogFormatText},
		{name: "json from env, case-insensitive", env: " JSON ", want: LogFormatJSON},
		{name: "json from file", fileYAML: "log:\n  format: json\n", want: LogFormatJSON},
		{name: "env beats file", env: "text", fileYAML: "log:\n  format: json\n", want: LogFormatText},
		{name: "invalid format", env: "xml", wantErr: "Invalid log format"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(envMesheryURL, "")
			t.Setenv(envMesheryToken, "")
			t.Setenv(envLogLevel, "")
			t.Setenv(envLogFormat, "")
			t.Setenv(envLogFormat, tt.env)
			clearTransportEnv(t)

			path := ""
			if tt.fileYAML != "" {
				path = writeConfigFile(t, tt.fileYAML)
			}

			cfg, err := Load(path, Flags{})
			if tt.wantErr != "" {
				if err == nil || errors.GetSDescription(err) != tt.wantErr {
					t.Fatalf("want error %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Log.Format != tt.want {
				t.Errorf("Log.Format = %q, want %q", cfg.Log.Format, tt.want)
			}
		})
	}
}
