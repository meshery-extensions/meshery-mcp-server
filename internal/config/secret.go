package config

import (
	"encoding/json"
	"fmt"
)

const redacted = "[REDACTED]"

// Secret redacts itself when printed or JSON-encoded; use string(s) only
// where the value is needed.
type Secret string

func (s Secret) String() string {
	if s == "" {
		return ""
	}
	return redacted
}

// %#v ignores String.
func (s Secret) GoString() string {
	return fmt.Sprintf("%q", s.String())
}

func (s Secret) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}
