package meshery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

// Client handles REST API communications with a running Meshery Server instance.
type Client struct {
	cfg  Config
	http *http.Client
}

// NewClient initializes and validates a new Meshery REST Client with the given configuration.
func NewClient(cfg Config) (*Client, error) {
	u, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("base URL scheme must be http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("base URL host cannot be empty")
	}
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("timeout must be greater than zero")
	}
	if cfg.RetryCount < 0 {
		return nil, fmt.Errorf("retry_count cannot be negative")
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: cfg.Timeout,
		}
	}

	return &Client{
		cfg:  cfg,
		http: httpClient,
	}, nil
}

// do is the core transport method executing HTTP calls with headers, retries, serialization, and error mapping.
func (c *Client) do(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	reqBody any,
	respBody any,
) error {
	u, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}
	u = u.JoinPath(path)
	if query != nil {
		u.RawQuery = query.Encode()
	}
	fullURL := u.String()

	var payload []byte
	if reqBody != nil {
		payload, err = json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	var lastErr error
	for attempt := 0; attempt <= c.cfg.RetryCount; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * 100 * time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		var bodyReader io.Reader
		if payload != nil {
			bodyReader = bytes.NewReader(payload)
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Accept", "application/json")
		if c.cfg.UserAgent != "" {
			req.Header.Set("User-Agent", c.cfg.UserAgent)
		}
		if reqBody != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if c.cfg.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			if isRetryableErr(err) {
				continue
			}
			return err
		}

		respData, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		if resp.StatusCode >= 400 {
			apiErr := parseAPIError(resp.StatusCode, method, fullURL, respData)
			if isRetryableStatusCode(resp.StatusCode) {
				lastErr = apiErr
				continue
			}
			return apiErr
		}

		if resp.StatusCode == http.StatusNoContent || len(respData) == 0 {
			return nil
		}

		if respBody != nil {
			if err := json.Unmarshal(respData, respBody); err != nil {
				return fmt.Errorf("failed to unmarshal response: %w", err)
			}
		}

		return nil
	}

	return lastErr
}

// parseAPIError constructs an APIError from an HTTP error status code and body bytes.
func parseAPIError(statusCode int, method, reqURL string, body []byte) *APIError {
	msg := string(body)

	var errPayload struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(body, &errPayload); err == nil {
		if errPayload.Message != "" {
			msg = errPayload.Message
		} else if errPayload.Error != "" {
			msg = errPayload.Error
		}
	}

	return &APIError{
		StatusCode: statusCode,
		Method:     method,
		URL:        reqURL,
		Message:    msg,
		RawBody:    body,
	}
}

// isRetryableErr classifies whether a network error should trigger a retry attempt.
func isRetryableErr(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
}

// isRetryableStatusCode classifies whether an HTTP status code should trigger a retry attempt.
func isRetryableStatusCode(statusCode int) bool {
	return statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout ||
		statusCode == http.StatusTooManyRequests
}
