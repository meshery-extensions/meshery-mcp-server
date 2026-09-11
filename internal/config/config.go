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
	"net/url"
	"os"
)

const (
	// DefaultMeshServerURL is the default base URL of the Meshery Server REST API.
	DefaultMeshServerURL = "http://localhost:9081"
	// DefaultHTTPAddr is the default address for the HTTP/SSE transport.
	DefaultHTTPAddr = "0.0.0.0:8080"
)

// Config holds runtime configuration for the Meshery MCP server.
type Config struct {
	// MeshServerURL is the base URL of the Meshery Server REST API.
	MeshServerURL string
	// MeshAPIToken is an optional token used to authenticate with the Meshery Server API.
	MeshAPIToken string
	// Transport selects the MCP transport: stdio, http, or sse.
	Transport string
	// HTTPAddr is the listen address for the http/sse transport.
	HTTPAddr string
}

// Load reads configuration from the environment, applying defaults where unset.
func Load() *Config {
	httpAddr := envOr("MESHERY_MCP_HTTP_ADDR", DefaultHTTPAddr)
	// Render and some PaaS inject PORT. Honor it if HTTPAddr is default and PORT is set.
	if httpAddr == DefaultHTTPAddr {
		if port := os.Getenv("PORT"); port != "" {
			httpAddr = "0.0.0.0:" + port
		}
	}
	return &Config{
		MeshServerURL: envOr("MESHERY_SERVER_URL", DefaultMeshServerURL),
		MeshAPIToken:  os.Getenv("MESHERY_API_TOKEN"),
		Transport:     envOr("MESHERY_MCP_TRANSPORT", "stdio"),
		HTTPAddr:      httpAddr,
	}
}

// RedactedURL returns the Meshery Server URL with any userinfo, query, and
// fragment components removed, suitable for logging.
func (c *Config) RedactedURL() string {
	if c.MeshServerURL == "" {
		return "<unset>"
	}
	u, err := url.Parse(c.MeshServerURL)
	if err != nil {
		return "<invalid>"
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

// envOr returns the value of the environment variable key, or fallback when
// the variable is unset or empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
