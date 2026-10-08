// Package security provides response boundary secret sanitization and redaction engines
// to prevent credentials, bearer tokens, API keys, and session secrets from leaking through MCP tool outputs.
package security

import (
	"encoding/json"
	"reflect"
	"strings"
)

// RedactedPlaceholder is the standardized replacement string for sensitive values.
const RedactedPlaceholder = "[REDACTED_SECRET]"

// CircularPlaceholder is the replacement string used when a circular/cyclic data structure is detected.
const CircularPlaceholder = "[CIRCULAR_REFERENCE]"

const maxRecursionDepth = 64

// Exact sensitive key names that trigger redaction, including Meshery/Kubeconfig secret data keys.
var exactSensitiveKeys = map[string]bool{
	"token":                      true,
	"password":                   true,
	"passwd":                     true,
	"secret":                     true,
	"private_key":                true,
	"privatekey":                 true,
	"kubeconfig":                 true,
	"auth_token":                 true,
	"authtoken":                  true,
	"api_key":                    true,
	"apikey":                     true,
	"access_token":               true,
	"accesstoken":                true,
	"refresh_token":              true,
	"refreshtoken":               true,
	"id_token":                   true,
	"idtoken":                    true,
	"pass":                       true,
	"session":                    true,
	"session_cookie":             true,
	"session_id":                 true,
	"sessionid":                  true,
	"authorization":              true,
	"client-key-data":            true,
	"client_key_data":            true,
	"client_secret":              true,
	"clientsecret":               true,
	"certificate-authority-data": true,
	"certificate_authority_data": true,
	"data":                       true,
	"stringdata":                 true,
	"string_data":                true,
	"aws_secret_access_key":      true,
	"aws_access_key_id":          true,
	"gcp_service_account":        true,
	"credential":                 true,
	"credentials":                true,
}

// Exceptions that must NOT be redacted despite containing common substrings.
var allowedExceptions = map[string]bool{
	"secretary":     true,
	"author":        true,
	"authority":     true,
	"authorized":    true,
	"token_type":    true,
	"token_count":   true,
	"tokencount":    true,
	"passthrough":   true,
	"pass_through":  true,
	"password_hint": true,
}

// SanitizeMap deeply copies and redacts sensitive information from map hierarchies.
// It tracks visited pointer references to prevent infinite recursion on cyclic data structures.
func SanitizeMap(input map[string]interface{}) (res map[string]interface{}) {
	defer func() {
		if r := recover(); r != nil {
			res = map[string]interface{}{"error": "[REDACTED_DUE_TO_SANITY_RECOVERY]"}
		}
	}()
	visited := make(map[uintptr]bool)
	return sanitizeMapInternal(input, visited, 0)
}

func sanitizeMapInternal(input map[string]interface{}, visited map[uintptr]bool, depth int) map[string]interface{} {
	if input == nil {
		return nil
	}
	if depth > maxRecursionDepth {
		return map[string]interface{}{"error": CircularPlaceholder}
	}

	valPtr := reflect.ValueOf(input).Pointer()
	if valPtr != 0 {
		if visited[valPtr] {
			return map[string]interface{}{"cycle": CircularPlaceholder}
		}
		visited[valPtr] = true
		defer delete(visited, valPtr)
	}

	result := make(map[string]interface{}, len(input))

	for k, v := range input {
		if isSensitiveKey(k) {
			result[k] = RedactedPlaceholder
			continue
		}

		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = sanitizeMapInternal(val, visited, depth+1)
		case []interface{}:
			result[k] = sanitizeSliceInternal(val, visited, depth+1)
		case string:
			result[k] = SanitizeString(val)
		default:
			result[k] = v
		}
	}

	return result
}

// SanitizeJSON parses raw JSON, scrubs sensitive keys, and returns sanitized JSON bytes.
func SanitizeJSON(rawJSON []byte) (res []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			res = []byte(RedactedPlaceholder)
			err = nil
		}
	}()
	if len(rawJSON) == 0 {
		return rawJSON, nil
	}

	var data interface{}
	if err := json.Unmarshal(rawJSON, &data); err != nil {
		sanitizedStr := SanitizeString(string(rawJSON))
		return []byte(sanitizedStr), nil
	}

	visited := make(map[uintptr]bool)
	sanitizedData := sanitizeValueInternal(data, visited, 0)
	return json.Marshal(sanitizedData)
}

// SanitizeString scrubs sensitive token or credential patterns from error messages and log outputs.
// If input is valid JSON, it routes through SanitizeJSON. Malformed braced input falls back safely without recursion.
func SanitizeString(input string) (res string) {
	defer func() {
		if r := recover(); r != nil {
			res = RedactedPlaceholder
		}
	}()
	if input == "" {
		return input
	}

	trimmed := strings.TrimSpace(input)
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		if json.Valid([]byte(trimmed)) {
			if sanitizedBytes, err := SanitizeJSON([]byte(trimmed)); err == nil {
				return string(sanitizedBytes)
			}
		}
	}

	result := input

	for key := range exactSensitiveKeys {
		// Exclude generic words from unquoted free-text matching to prevent false positives
		if key == "data" || key == "session" || key == "pass" {
			continue
		}

		patterns := []string{
			"\"" + key + "\": \"",
			"\"" + key + "\":\"",
			"\"" + key + "\": ",
			"\"" + key + "\":",
			"'" + key + "': '",
			"'" + key + "':",
			key + "= ",
			key + "=",
			key + ": ",
			key + ":",
		}
		for _, pattern := range patterns {
			searchIdx := 0
			for {
				if searchIdx >= len(result) {
					break
				}
				lowerResult := strings.ToLower(result[searchIdx:])
				idx := strings.Index(lowerResult, strings.ToLower(pattern))
				if idx == -1 {
					break
				}
				realIdx := searchIdx + idx

				// Skip matches preceded by word characters (letter, digit, underscore, hyphen)
				if realIdx > 0 {
					prev := result[realIdx-1]
					if (prev >= 'a' && prev <= 'z') || (prev >= 'A' && prev <= 'Z') || (prev >= '0' && prev <= '9') || prev == '_' || prev == '-' {
						searchIdx = realIdx + len(pattern)
						continue
					}
				}

				valStart := realIdx + len(pattern)
				remainder := result[valStart:]

				hasQuote := false
				quoteChar := byte(0)
				if strings.HasSuffix(pattern, "\"") {
					hasQuote = true
					quoteChar = '"'
				} else if strings.HasSuffix(pattern, "'") {
					hasQuote = true
					quoteChar = '\''
				} else if len(remainder) > 0 && (remainder[0] == '"' || remainder[0] == '\'') {
					hasQuote = true
					quoteChar = remainder[0]
					valStart++
					remainder = result[valStart:]
				}

				prefix := ""
				if strings.HasPrefix(strings.ToLower(remainder), "bearer ") {
					prefix = "Bearer "
					valStart += len("bearer ")
				}

				var endIdx int
				if hasQuote && quoteChar != 0 {
					quoteIdx := strings.IndexByte(result[valStart:], quoteChar)
					if quoteIdx != -1 {
						endIdx = valStart + quoteIdx
					} else {
						endIdx = len(result)
					}
				} else {
					valEnd := strings.IndexAny(result[valStart:], " \t\n\r\",;}")
					if valEnd == -1 {
						endIdx = len(result)
					} else {
						endIdx = valStart + valEnd
					}
				}

				replacement := prefix + RedactedPlaceholder
				if hasQuote {
					if endIdx < len(result) && result[endIdx] == quoteChar {
						result = result[:valStart] + replacement + result[endIdx:]
					} else {
						result = result[:valStart] + replacement + string(quoteChar) + result[endIdx:]
					}
				} else {
					result = result[:valStart] + replacement + result[endIdx:]
				}
				searchIdx = valStart + len(replacement) + 1
			}
		}
	}
	return result
}

// Helper: delimiter-bounded key matching to prevent over-redaction (e.g., secretary -> NOT redacted).
func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	if allowedExceptions[lower] {
		return false
	}
	if exactSensitiveKeys[lower] {
		return true
	}

	tokens := strings.FieldsFunc(lower, func(r rune) bool {
		return r == '_' || r == '-' || r == '.' || r == '@' || r == '/' || r == ' '
	})

	for _, token := range tokens {
		if exactSensitiveKeys[token] && !allowedExceptions[lower] {
			return true
		}
	}
	return false
}

func sanitizeSliceInternal(s []interface{}, visited map[uintptr]bool, depth int) []interface{} {
	if s == nil {
		return nil
	}
	if depth > maxRecursionDepth {
		return []interface{}{CircularPlaceholder}
	}

	valPtr := reflect.ValueOf(s).Pointer()
	if valPtr != 0 {
		if visited[valPtr] {
			return []interface{}{CircularPlaceholder}
		}
		visited[valPtr] = true
		defer delete(visited, valPtr)
	}

	result := make([]interface{}, len(s))
	for i, item := range s {
		switch child := item.(type) {
		case map[string]interface{}:
			result[i] = sanitizeMapInternal(child, visited, depth+1)
		case []interface{}:
			result[i] = sanitizeSliceInternal(child, visited, depth+1)
		case string:
			result[i] = SanitizeString(child)
		default:
			result[i] = item
		}
	}

	return result
}

func sanitizeValueInternal(v interface{}, visited map[uintptr]bool, depth int) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return sanitizeMapInternal(val, visited, depth+1)
	case []interface{}:
		return sanitizeSliceInternal(val, visited, depth+1)
	case string:
		return SanitizeString(val)
	default:
		return v
	}
}
