package meshery

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"strings"
)

// Category sentinels for errors.Is.
var (
	ErrNotFound   = stderrors.New("meshery: not found")
	ErrConflict   = stderrors.New("meshery: conflict")
	ErrValidation = stderrors.New("meshery: validation failed")
	ErrServer     = stderrors.New("meshery: server error")
)

// APIError is a typed non-2xx response. Path never includes the query string,
// which may carry secrets. Message is the server's own message.
type APIError struct {
	Status   int
	Method   string
	Path     string
	Message  string
	Code     string
	Causes   []string
	Remedies []string
	Kind     error
}

func (e *APIError) Error() string {
	msg := strings.TrimSpace(e.Message)
	if msg == "" {
		msg = http.StatusText(e.Status)
	}
	if e.Code != "" {
		return fmt.Sprintf("%s %s failed: HTTP %d (%s): %s", e.Method, e.Path, e.Status, e.Code, msg)
	}
	return fmt.Sprintf("%s %s failed: HTTP %d: %s", e.Method, e.Path, e.Status, msg)
}

// Unwrap exposes the category so errors.Is(err, ErrNotFound) works.
func (e *APIError) Unwrap() error { return e.Kind }

func (e *APIError) Is(target error) bool { return target != nil && target == e.Kind }

// apiError builds an APIError and removes the token captured for this request
// from every server-supplied field retained in the error.
func (c *Client) apiError(method, path string, status int, body []byte, token string) *APIError {
	return newAPIError(method, path, status, body, token)
}

// apiErrorBody is the shape Meshery Server writes (writeMeshkitError and
// writeJSONError).
type apiErrorBody struct {
	Error                string   `json:"error"`
	Code                 string   `json:"code"`
	ProbableCause        []string `json:"probableCause"`
	SuggestedRemediation []string `json:"suggestedRemediation"`
}

func newAPIError(method, path string, status int, body []byte, token string) *APIError {
	e := &APIError{
		Status: status,
		Method: method,
		Path:   pathWithoutQuery(path),
		Kind:   kindForStatus(status),
	}
	var parsed apiErrorBody
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Error != "" {
		e.Message = parsed.Error
		e.Code = parsed.Code
		e.Causes = parsed.ProbableCause
		e.Remedies = parsed.SuggestedRemediation
	} else {
		e.Message = truncate(redact(token, string(body)), 512)
		return e
	}
	if token == "" {
		return e
	}
	e.Message = strings.ReplaceAll(e.Message, token, "[REDACTED]")
	e.Code = strings.ReplaceAll(e.Code, token, "[REDACTED]")
	for i := range e.Causes {
		e.Causes[i] = strings.ReplaceAll(e.Causes[i], token, "[REDACTED]")
	}
	for i := range e.Remedies {
		e.Remedies[i] = strings.ReplaceAll(e.Remedies[i], token, "[REDACTED]")
	}
	return e
}

func kindForStatus(status int) error {
	switch status {
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusConflict:
		return ErrConflict
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return ErrValidation
	default:
		if status >= 500 {
			return ErrServer
		}
		return nil
	}
}

func pathWithoutQuery(path string) string {
	if i := strings.IndexByte(path, '?'); i >= 0 {
		return path[:i]
	}
	return path
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// redact removes a credential from server text before it is put in an error.
func redact(token, text string) string {
	text = strings.TrimSpace(text)
	if token == "" {
		return text
	}
	return strings.ReplaceAll(text, token, "[REDACTED]")
}
