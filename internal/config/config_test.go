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

// isolateEnv clears the same table load binds, so new settings need no test
// changes to stay isolated.
func isolateEnv(t *testing.T) {
	t.Helper()
	t.Setenv(envMesheryToken, "")
	for _, b := range envBindings {
		t.Setenv(b.env, "")
	}
}

func setEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for name, value := range env {
		t.Setenv(name, value)
	}
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want error %s, got nil", code)
	}
	if got := errors.GetCode(err); got != code {
		t.Fatalf("error code = %q, want %q (error: %v)", got, code, err)
	}
}

func TestLoad(t *testing.T) {
	const fileYAML = "meshery:\n  url: https://file.example:9081\nlog:\n  level: warn\n"

	tests := []struct {
		name      string
		env       map[string]string
		fileYAML  string // "" = no config file
		path      string // explicit path; overrides fileYAML
		wantURL   string
		wantToken Secret
		wantLevel string
		wantCode  string // "" = expect success
	}{
		{name: "defaults only", wantURL: defaultMesheryURL, wantLevel: LogLevelInfo},
		{name: "env overrides default",
			env:     map[string]string{envMesheryURL: "https://env.example", envMesheryToken: "env-token", envLogLevel: "debug"},
			wantURL: "https://env.example", wantToken: "env-token", wantLevel: LogLevelDebug},
		{name: "file only", fileYAML: fileYAML,
			wantURL: "https://file.example:9081", wantLevel: LogLevelWarn},
		{name: "env beats file per key", fileYAML: fileYAML,
			env:     map[string]string{envMesheryURL: "https://env.example"},
			wantURL: "https://env.example", wantLevel: LogLevelWarn},
		{name: "empty env var counts as unset", fileYAML: fileYAML,
			env:     map[string]string{envMesheryURL: ""},
			wantURL: "https://file.example:9081", wantLevel: LogLevelWarn},
		{name: "log level is case-insensitive", env: map[string]string{envLogLevel: " DEBUG "},
			wantURL: defaultMesheryURL, wantLevel: LogLevelDebug},
		{name: "repo config_example.yml loads", path: filepath.Join("..", "..", "config_example.yml"),
			wantURL: defaultMesheryURL, wantLevel: LogLevelInfo},
		{name: "invalid scheme", env: map[string]string{envMesheryURL: "ftp://x"}, wantCode: ErrInvalidMesheryURLCode},
		{name: "missing scheme", env: map[string]string{envMesheryURL: "localhost:9081"}, wantCode: ErrInvalidMesheryURLCode},
		{name: "unparsable URL", env: map[string]string{envMesheryURL: "http://u:p@host:90x81"}, wantCode: ErrInvalidMesheryURLCode},
		{name: "invalid log level", env: map[string]string{envLogLevel: "verbose"}, wantCode: ErrInvalidLogLevelCode},
		{name: "missing file", path: filepath.Join(t.TempDir(), "nope.yaml"), wantCode: ErrReadConfigFileCode},
		{name: "file that is not YAML", fileYAML: "meshery: [", wantCode: ErrReadConfigFileCode},

		// Token
		{name: "token from the environment", env: map[string]string{envMesheryToken: "env-token"},
			wantURL: defaultMesheryURL, wantToken: "env-token", wantLevel: LogLevelInfo},
		{name: "token under meshery in file", fileYAML: "meshery:\n  token: file-token\n", wantCode: ErrTokenInConfigFileCode},
		{name: "token under auth in file", fileYAML: "auth:\n  token: file-token\n", wantCode: ErrTokenInConfigFileCode},
		{name: "token under transport.stdio in file", fileYAML: "transport:\n  stdio:\n    token: file-token\n",
			wantCode: ErrTokenInConfigFileCode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateEnv(t)
			setEnv(t, tt.env)

			path := tt.path
			if path == "" && tt.fileYAML != "" {
				path = writeConfigFile(t, tt.fileYAML)
			}

			cfg, err := Load(path, Flags{})
			if tt.wantCode != "" {
				wantCode(t, err, tt.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Meshery.URL != tt.wantURL {
				t.Errorf("URL = %q, want %q", cfg.Meshery.URL, tt.wantURL)
			}
			if cfg.Transport.Stdio.Token != tt.wantToken {
				t.Errorf("Stdio.Token = %q, want %q", string(cfg.Transport.Stdio.Token), string(tt.wantToken))
			}
			if cfg.Log.Level != tt.wantLevel {
				t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, tt.wantLevel)
			}
		})
	}
}

// load, not Load, so the http settings can be tested.
func TestLoad_Transport(t *testing.T) {
	str := func(s string) *string { return &s }
	num := func(n int) *int { return &n }
	stdio := func(provider string, token Secret) TransportConfig {
		return TransportConfig{Mode: TransportStdio, Stdio: StdioConfig{Provider: provider, Token: token},
			HTTP: HTTPConfig{Host: defaultHTTPHost, Port: defaultHTTPPort}}
	}
	http := func(host string, port int) TransportConfig {
		return TransportConfig{Mode: TransportHTTP, HTTP: HTTPConfig{Host: host, Port: port}}
	}

	tests := []struct {
		name     string
		env      map[string]string
		fileYAML string
		flags    Flags
		want     TransportConfig
		wantCode string
	}{
		// stdio identity
		{name: "defaults to stdio with the Local provider",
			want: stdio(LocalProvider, "")},
		{name: "a token defaults to the Meshery provider", env: map[string]string{envMesheryToken: "env-token"},
			want: stdio(DefaultRemoteProvider, "env-token")},
		{name: "provider from env, trimmed", env: map[string]string{envMesheryToken: "env-token", envProvider: " Layer5 "},
			want: stdio("Layer5", "env-token")},
		{name: "provider from file", env: map[string]string{envMesheryToken: "env-token"},
			fileYAML: "transport:\n  stdio:\n    provider: Layer5\n",
			want:     stdio("Layer5", "env-token")},
		{name: "env provider beats file", env: map[string]string{envMesheryToken: "env-token", envProvider: "Meshery"},
			fileYAML: "transport:\n  stdio:\n    provider: Layer5\n",
			want:     stdio("Meshery", "env-token")},
		{name: "remote provider without a token", env: map[string]string{envProvider: "Meshery"},
			want: stdio("Meshery", "")},
		{name: "token with the Local provider", env: map[string]string{envMesheryToken: "env-token", envProvider: "local"},
			wantCode: ErrTokenWithLocalProviderCode},
		{name: "token with the legacy None provider", env: map[string]string{envMesheryToken: "env-token", envProvider: "None"},
			wantCode: ErrTokenWithLocalProviderCode},
		{name: "stdio ignores host", flags: Flags{Host: str("0.0.0.0")},
			want: TransportConfig{Mode: TransportStdio, Stdio: StdioConfig{Provider: LocalProvider},
				HTTP: HTTPConfig{Host: "0.0.0.0", Port: defaultHTTPPort}}},

		// http
		{name: "http from env, case-insensitive", env: map[string]string{envTransportMode: "HTTP", envHTTPPort: "9000"},
			want: http(defaultHTTPHost, 9000)},
		{name: "http from file", fileYAML: "transport:\n  mode: http\n  http:\n    host: localhost\n    port: 7000\n",
			want: http("localhost", 7000)},
		{name: "flags beat env", env: map[string]string{envTransportMode: "stdio", envHTTPPort: "9000"},
			flags: Flags{Transport: str("http"), Port: num(8181)},
			want:  http(defaultHTTPHost, 8181)},
		{name: "IPv6 loopback", flags: Flags{Transport: str("http"), Host: str("::1")},
			want: http("::1", defaultHTTPPort)},
		{name: "host is lowercased", flags: Flags{Transport: str("http"), Host: str("LocalHost")},
			want: http("localhost", defaultHTTPPort)},
		{name: "http clears the stdio identity", env: map[string]string{envTransportMode: "http", envProvider: "Layer5"},
			want: http(defaultHTTPHost, defaultHTTPPort)},
		{name: "http refuses the token", flags: Flags{Transport: str("http")}, env: map[string]string{envMesheryToken: "env-token"},
			wantCode: ErrTokenNotAllowedForHTTPCode},
		{name: "non-loopback address", flags: Flags{Transport: str("http"), Host: str("0.0.0.0")},
			wantCode: ErrNonLoopbackHostCode},
		{name: "hostname that is not localhost", flags: Flags{Transport: str("http"), Host: str("example.com")},
			wantCode: ErrNonLoopbackHostCode},
		{name: "port out of range", flags: Flags{Transport: str("http"), Port: num(70000)},
			wantCode: ErrInvalidPortCode},
		{name: "port not a number", env: map[string]string{envTransportMode: "http", envHTTPPort: "eighty"},
			wantCode: ErrDecodeConfigCode},
		{name: "sse is not supported", flags: Flags{Transport: str("sse")},
			wantCode: ErrInvalidTransportCode},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateEnv(t)
			setEnv(t, tt.env)

			path := ""
			if tt.fileYAML != "" {
				path = writeConfigFile(t, tt.fileYAML)
			}

			cfg, err := load(path, tt.flags)
			if tt.wantCode != "" {
				wantCode(t, err, tt.wantCode)
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

func TestLoad_HTTPRequiresAuthorization(t *testing.T) {
	isolateEnv(t)
	http := "http"

	_, err := Load("", Flags{Transport: &http})
	wantCode(t, err, ErrHTTPAuthorizationRequiredCode)
}

func TestLoad_LogFormat(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		fileYAML string
		want     string
		wantCode string
	}{
		{name: "defaults to text", want: LogFormatText},
		{name: "json from env, case-insensitive", env: " JSON ", want: LogFormatJSON},
		{name: "json from file", fileYAML: "log:\n  format: json\n", want: LogFormatJSON},
		{name: "env beats file", env: "text", fileYAML: "log:\n  format: json\n", want: LogFormatText},
		{name: "invalid format", env: "xml", wantCode: ErrInvalidLogFormatCode},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateEnv(t)
			t.Setenv(envLogFormat, tt.env)

			path := ""
			if tt.fileYAML != "" {
				path = writeConfigFile(t, tt.fileYAML)
			}

			cfg, err := Load(path, Flags{})
			if tt.wantCode != "" {
				wantCode(t, err, tt.wantCode)
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

func TestInvalidURLErrorDoesNotLeakCredentials(t *testing.T) {
	isolateEnv(t)
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
	http := "http"
	tests := []struct {
		name  string
		file  string
		flags Flags
		env   map[string]string
	}{
		{name: "in config file", file: "meshery:\n  token: file-secret\n"},
		{name: "with the http transport", flags: Flags{Transport: &http}},
		{name: "with the Local provider", env: map[string]string{envProvider: "Local"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateEnv(t)
			t.Setenv(envMesheryToken, "super-secret")
			setEnv(t, tt.env)
			path := ""
			if tt.file != "" {
				path = writeConfigFile(t, tt.file)
			}

			_, err := load(path, tt.flags)
			if err == nil {
				t.Fatal("expected an error")
			}
			for _, secret := range []string{"super-secret", "file-secret"} {
				if strings.Contains(err.Error(), secret) {
					t.Errorf("token leaked into error: %v", err)
				}
			}
		})
	}
}

// Built at run time: gosec reports a URL literal with a password (G101).
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
		Meshery:   MesheryConfig{URL: "http://x"},
		Transport: TransportConfig{Stdio: StdioConfig{Token: "super-secret"}},
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	outputs := map[string]string{"json": string(b)}
	for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
		outputs[format+" config"] = fmt.Sprintf(format, cfg)
		outputs[format+" token"] = fmt.Sprintf(format, cfg.Transport.Stdio.Token)
	}
	for name, out := range outputs {
		if strings.Contains(out, "super-secret") {
			t.Errorf("%s leaked the token: %s", name, out)
		}
	}
	if Secret("").String() != "" {
		t.Error("an empty secret should print as empty, so a missing token stays visible")
	}
}
