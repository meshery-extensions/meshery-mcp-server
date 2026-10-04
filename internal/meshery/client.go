package meshery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultServerURL = "http://localhost:9081"

// Client is a client for communicating with a Meshery server.
type Client struct {
	BaseURL  string
	APIToken string
	HTTP     *http.Client
}

// NewClient creates a Meshery API client.
//
// Configuration:
//   - MESHERY_SERVER_URL sets the Meshery server URL.
//   - MESHERY_API_TOKEN sets the provider/API token.
//   - If MESHERY_SERVER_URL is not set, localhost:9081 is used.
func NewClient() *Client {
	baseURL := os.Getenv("MESHERY_SERVER_URL")
	if baseURL == "" {
		baseURL = defaultServerURL
	}

	return &Client{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		APIToken: os.Getenv("MESHERY_API_TOKEN"),
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Do performs an HTTP request against Meshery.
func (c *Client) Do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var requestBody *bytes.Reader

	if body == nil {
		requestBody = bytes.NewReader(nil)
	} else {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}

		requestBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		c.BaseURL+path,
		requestBody,
	)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIToken)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("meshery request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("meshery API returned status %d", resp.StatusCode)
	}

	if result == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode Meshery response: %w", err)
	}

	return nil
}
