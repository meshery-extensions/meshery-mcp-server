package meshery

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"time"
)

// ClientConfig holds the target base URL and the filepath to auth credentials
type ClientConfig struct {
	ServerURL  string `json:"server_url"`
	ConfigPath string `json:"config_path"`
}

// authFile represents the typical layout of Meshery configuration tokens
type authFile struct {
	Token           string `json:"token"`
	MesheryProvider string `json:"meshery-provider"`
}

// NewAuthenticatedClient initializes an http.Client pre-loaded with session cookies
func NewAuthenticatedClient(cfg ClientConfig) (*http.Client, error) {
	// 1. Initialize a secure cookie jar to handle session state
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cookie jar: %w", err)
	}

	// 2. Instantiate the default client wrapper with a strict timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
		Jar:     jar,
	}

	// 3. If no config path is passed, fall back to a clean unauthenticated instance
	if cfg.ConfigPath == "" {
		return client, nil
	}

	// 4. Read and extract credentials from the auth file workspace
	data, err := os.ReadFile(cfg.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read authentication file at %s: %w", cfg.ConfigPath, err)
	}

	var auth authFile
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, fmt.Errorf("failed to parse auth JSON structures: %w", err)
	}

	// 5. Parse target server URL explicitly to map the cookies correctly
	parsedURL, err := url.Parse(cfg.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server base URL format '%s': %w", cfg.ServerURL, err)
	}

	// 6. Bind the meshery-token cookie to the target domain if present
	if auth.Token != "" {
		cookies := []*http.Cookie{
			{
				Name:  "meshery-token",
				Value: auth.Token,
				Path:  "/",
			},
		}
		client.Jar.SetCookies(parsedURL, cookies)
	}

	return client, nil
}
