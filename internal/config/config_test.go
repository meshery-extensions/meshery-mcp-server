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

import "testing"

// TestRedactedURL verifies that RedactedURL strips credentials, query
// parameters, and fragments from a variety of URL forms.
func TestRedactedURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "plain", raw: "http://localhost:9081", want: "http://localhost:9081"},
		{name: "with credentials", raw: "http://user:pass@localhost:9081", want: "http://localhost:9081"},
		{name: "with query token", raw: "http://localhost:9081/path?token=secret", want: "http://localhost:9081/path"},
		{name: "with fragment", raw: "http://localhost:9081#frag", want: "http://localhost:9081"},
		{name: "empty", raw: "", want: "<unset>"},
		{name: "invalid", raw: "://bad", want: "<invalid>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{MeshServerURL: tt.raw}
			if got := c.RedactedURL(); got != tt.want {
				t.Errorf("RedactedURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestLoadDefaults verifies that Load applies the default Meshery Server URL
// when environment variables are unset.
func TestLoadDefaults(t *testing.T) {
	t.Setenv("MESHERY_SERVER_URL", "")
	t.Setenv("MESHERY_API_TOKEN", "")
	cfg := Load()
	if cfg.MeshServerURL != DefaultMeshServerURL {
		t.Errorf("MeshServerURL = %q, want %q", cfg.MeshServerURL, DefaultMeshServerURL)
	}
}

// TestLoadFromEnvironment verifies that Load reads non-empty values from the
// environment instead of applying defaults.
func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("MESHERY_SERVER_URL", "https://example.meshery.io:9443")
	t.Setenv("MESHERY_API_TOKEN", "test-token")
	cfg := Load()
	if cfg.MeshServerURL != "https://example.meshery.io:9443" {
		t.Errorf("MeshServerURL = %q, want %q", cfg.MeshServerURL, "https://example.meshery.io:9443")
	}
	if cfg.MeshAPIToken != "test-token" {
		t.Errorf("MeshAPIToken = %q, want %q", cfg.MeshAPIToken, "test-token")
	}
}
