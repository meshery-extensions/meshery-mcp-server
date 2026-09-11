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

package server

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

func TestRegistryRegisterAll(t *testing.T) {
	s := mcpserver.NewMCPServer("test-server", "0.0.1")

	var calls []string
	registry := NewRegistry(
		RegistrantFunc(func(*mcpserver.MCPServer) error {
			calls = append(calls, "tools")
			return nil
		}),
		RegistrantFunc(func(*mcpserver.MCPServer) error {
			calls = append(calls, "resources")
			return nil
		}),
	)

	if err := registry.RegisterAll(s); err != nil {
		t.Fatalf("register all: %v", err)
	}

	expected := []string{"tools", "resources"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("expected calls %v, got %v", expected, calls)
	}
}

func TestRegistryRegisterAllStopsOnError(t *testing.T) {
	s := mcpserver.NewMCPServer("test-server", "0.0.1")
	expectedErr := errors.New("registration failed")

	var calls []string
	registry := NewRegistry(
		Named("tools", RegistrantFunc(func(*mcpserver.MCPServer) error {
			calls = append(calls, "tools")
			return nil
		})),
		Named("resources", RegistrantFunc(func(*mcpserver.MCPServer) error {
			calls = append(calls, "resources")
			return expectedErr
		})),
		Named("prompts", RegistrantFunc(func(*mcpserver.MCPServer) error {
			calls = append(calls, "prompts")
			return nil
		})),
	)

	err := registry.RegisterAll(s)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
	if !strings.Contains(err.Error(), "resources") {
		t.Fatalf("expected error to name the failing surface, got %q", err)
	}

	expected := []string{"tools", "resources"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("expected calls %v, got %v", expected, calls)
	}
}

func TestRegistryRegisterAllNamesUnlabeledRegistrantByType(t *testing.T) {
	s := mcpserver.NewMCPServer("test-server", "0.0.1")
	expectedErr := errors.New("registration failed")

	registry := NewRegistry(RegistrantFunc(func(*mcpserver.MCPServer) error {
		return expectedErr
	}))

	err := registry.RegisterAll(s)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
	if !strings.Contains(err.Error(), "RegistrantFunc") {
		t.Fatalf("expected error to name the registrant type, got %q", err)
	}
}
