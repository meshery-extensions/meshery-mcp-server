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

func TestMaskToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{name: "empty token", token: "", want: "<not set>"},
		{name: "short token", token: "abc", want: "***"},
		{name: "exactly 8 chars", token: "abcd1234", want: "abc***234"},
		{name: "long token", token: "abc123xyz789def", want: "abc***def"},
		{name: "7 char token", token: "abcdefg", want: "***"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskToken(tt.token); got != tt.want {
				t.Errorf("MaskToken(%q) = %q, want %q", tt.token, got, tt.want)
			}
		})
	}
}

func TestMaskedContext(t *testing.T) {
	ctx := Context{
		Server: "http://localhost:9081",
		Token:  "secret-token-value",
	}

	masked := MaskedContext(ctx)

	if masked.Server != ctx.Server {
		t.Errorf("Server should not be masked: got %q, want %q", masked.Server, ctx.Server)
	}

	if masked.Token == ctx.Token {
		t.Error("Token should be masked")
	}

	if masked.Token != "sec***lue" {
		t.Errorf("MaskedContext token = %q, want %q", masked.Token, "sec***lue")
	}
}
