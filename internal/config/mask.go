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

// MaskToken returns a masked version of the token for safe logging.
// Shows first 3 and last 3 characters with asterisks in between.
// Short tokens are fully masked.
//
// Examples:
//
//	"abc123xyz789" -> "abc***789"
//	"short" -> "***"
//	"" -> "<not set>"
func MaskToken(token string) string {
	if token == "" {
		return "<not set>"
	}

	const minVisibleLength = 8 // Minimum length to show partial token

	if len(token) < minVisibleLength {
		return "***"
	}

	return token[:3] + "***" + token[len(token)-3:]
}

// MaskedContext returns a Context with the token masked for safe logging.
func MaskedContext(ctx Context) Context {
	return Context{
		Server: ctx.Server,
		Token:  MaskToken(ctx.Token),
	}
}
