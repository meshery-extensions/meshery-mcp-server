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

	"github.com/meshery/meshkit/utils"
)

// Client handles REST API communications with a running Meshery Server instance.
type Client struct {
	cfg     Config
	baseURL *url.URL
	http    *http.Client
}

// NewClient initializes and validates a new Meshery REST Client with the given configuration.
// If a custom HTTPClient is supplied in cfg, its Timeout property will be set to cfg.Timeout.
func NewClient(cfg Config) (*Client, error) {
	u, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, ErrInvalidBaseURL(err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, ErrInvalidBaseURL(fmt.Errorf("base URL scheme must be http or https, got %q", u.Scheme))
	}
	if u.Host == "" {
		return nil, ErrInvalidBaseURL(errors.New("base URL host cannot be empty"))
	}
	if cfg.Token != "" && u.Scheme == "http" {
		host := u.Hostname()
		if host != "localhost" && host != "127.0.0.1" && host != "::1" {
			return nil, ErrInvalidBaseURL(errors.New("bearer tokens require HTTPS unless using a loopback address"))
		}
	}
	if cfg.Timeout <= 0 {
		return nil, ErrInvalidTimeout(cfg.Timeout)
	}
	if cfg.RetryCount < 0 {
		return nil, ErrInvalidRetryCount(cfg.RetryCount)
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: cfg.Timeout,
		}
	} else {
		httpClient.Timeout = cfg.Timeout
	}

	return &Client{
		cfg:     cfg,
		baseURL: u,
		http:    httpClient,
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
	u := *c.baseURL
	reqURL := u.JoinPath(path)
	if query != nil {
		reqURL.RawQuery = query.Encode()
	}
	fullURL := reqURL.String()

	var payload []byte
	if reqBody != nil {
		var err error
		payload, err = json.Marshal(reqBody)
		if err != nil {
			return utils.ErrMarshal(err)
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
			return ErrHTTPRequest(err, method, fullURL)
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
			meshErr := ErrHTTPRequest(err, method, fullURL)
			if isRetryableErr(err) && isRetryableMethod(method) {
				lastErr = meshErr
				continue
			}
			return meshErr
		}

		respData, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}

		if resp.StatusCode >= 400 {
			msg := string(respData)
			var errPayload struct {
				Message string `json:"message"`
				Error   string `json:"error"`
			}
			if err := json.Unmarshal(respData, &errPayload); err == nil {
				if errPayload.Message != "" {
					msg = errPayload.Message
				} else if errPayload.Error != "" {
					msg = errPayload.Error
				}
			}
			apiErr := ErrAPIResponse(resp.StatusCode, method, fullURL, msg)
			if isRetryableStatusCode(resp.StatusCode) && isRetryableMethod(method) {
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
				return utils.ErrUnmarshal(err)
			}
		}

		return nil
	}

	return lastErr
}

// isRetryableMethod classifies whether an HTTP method is safe to retry automatically.
func isRetryableMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
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
