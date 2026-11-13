package tools

import (
	"fmt"

	"github.com/meshery/meshkit/errors"
)

const (
	ErrServerRequiredCode    = "meshery-mcp-server-1000"
	ErrRegistryRequiredCode  = "meshery-mcp-server-1001"
	ErrRequiredArgumentCode  = "meshery-mcp-server-1002"
	ErrNegativeArgumentCode  = "meshery-mcp-server-1003"
	ErrSearchComponentsCode  = "meshery-mcp-server-1004"
	ErrGetComponentCode      = "meshery-mcp-server-1005"
	ErrListModelsCode        = "meshery-mcp-server-1006"
	ErrGetModelCode          = "meshery-mcp-server-1007"
	ErrListRelationshipsCode = "meshery-mcp-server-1008"
)

// fromRegistry returns a registry error to the AI. A MeshKit error already
// describes the failure with its own code, cause and remedy, so it is
// returned unchanged; wrapping it would repeat and contradict that text. Any
// other error is wrapped with wrap, so the AI still gets a cause and a remedy.
func fromRegistry(err error, wrap func(error) error) error {
	if _, ok := errors.Is(err); ok {
		return err
	}
	return wrap(err)
}

// ErrServerRequired is returned when Register is called without an MCP server.
func ErrServerRequired() error {
	return errors.New(ErrServerRequiredCode, errors.Alert,
		[]string{"MCP server is required to register tools"},
		[]string{"tools.Register was called with a nil *mcp.Server"},
		[]string{"The server was not constructed before registering tools"},
		[]string{"Construct the MCP server before calling tools.Register"})
}

// ErrRegistryRequired is returned when Register is called without a Registry.
func ErrRegistryRequired() error {
	return errors.New(ErrRegistryRequiredCode, errors.Alert,
		[]string{"Meshery registry is required to register tools"},
		[]string{"tools.Register was called with a nil Registry"},
		[]string{"The Meshery registry implementation was not constructed or not passed in"},
		[]string{"Pass a tools.Registry implementation to tools.Register"})
}

// ErrRequiredArgument is returned to the AI when a tool argument that must
// not be empty is empty. The input schema only checks that it is present.
func ErrRequiredArgument(argument string) error {
	return errors.New(ErrRequiredArgumentCode, errors.Alert,
		[]string{"Required tool argument is empty"},
		[]string{fmt.Sprintf("The %q argument must not be empty", argument)},
		[]string{"The tool was called with an empty value for a required argument"},
		[]string{fmt.Sprintf("Call the tool again with a non-empty %q", argument)})
}

// ErrNegativeArgument is returned to the AI when a numeric tool argument is negative.
func ErrNegativeArgument(argument string, value int) error {
	return errors.New(ErrNegativeArgumentCode, errors.Alert,
		[]string{"Tool argument must not be negative"},
		[]string{fmt.Sprintf("The %q argument is %d", argument, value)},
		[]string{"The tool was called with a negative number"},
		[]string{fmt.Sprintf("Call the tool again with %q set to 0 or more, or leave it out", argument)})
}

// ErrSearchComponents is returned when the registry fails to search components.
func ErrSearchComponents(err error) error {
	return errors.New(ErrSearchComponentsCode, errors.Alert,
		[]string{"Unable to search components"},
		[]string{fmt.Sprintf("Searching components failed: %v", err)},
		[]string{"The Meshery server returned an error or could not be reached"},
		[]string{"Check that the Meshery server is running and reachable, then try again"})
}

// ErrGetComponent is returned when the registry fails to get a component.
func ErrGetComponent(err error, id string) error {
	return errors.New(ErrGetComponentCode, errors.Alert,
		[]string{"Unable to get component"},
		[]string{fmt.Sprintf("Getting component %q failed: %v", id, err)},
		[]string{"The component does not exist, or the Meshery server returned an error or could not be reached"},
		[]string{"Check the component ID with search_components, and that the Meshery server is reachable"})
}

// ErrListModels is returned when the registry fails to list models.
func ErrListModels(err error) error {
	return errors.New(ErrListModelsCode, errors.Alert,
		[]string{"Unable to list models"},
		[]string{fmt.Sprintf("Listing models failed: %v", err)},
		[]string{"The Meshery server returned an error or could not be reached"},
		[]string{"Check that the Meshery server is running and reachable, then try again"})
}

// ErrGetModel is returned when the registry fails to get a model.
func ErrGetModel(err error, id string) error {
	return errors.New(ErrGetModelCode, errors.Alert,
		[]string{"Unable to get model"},
		[]string{fmt.Sprintf("Getting model %q failed: %v", id, err)},
		[]string{"The model does not exist, or the Meshery server returned an error or could not be reached"},
		[]string{"Check the model ID with list_models, and that the Meshery server is reachable"})
}

// ErrListRelationships is returned when the registry fails to list relationships.
func ErrListRelationships(err error) error {
	return errors.New(ErrListRelationshipsCode, errors.Alert,
		[]string{"Unable to list relationships"},
		[]string{fmt.Sprintf("Listing relationships failed: %v", err)},
		[]string{"The Meshery server returned an error or could not be reached"},
		[]string{"Check that the Meshery server is running and reachable, then try again"})
}
