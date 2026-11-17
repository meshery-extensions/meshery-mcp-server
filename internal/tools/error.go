package tools

import (
	"fmt"
	"strings"

	"github.com/meshery/meshkit/errors"
)

const (
	ErrServerRequiredCode    = "meshery-mcp-server-1000"
	ErrBackendRequiredCode   = "meshery-mcp-server-1001"
	ErrRequiredArgumentCode  = "meshery-mcp-server-1002"
	ErrNegativeArgumentCode  = "meshery-mcp-server-1003"
	ErrSearchComponentsCode  = "meshery-mcp-server-1004"
	ErrGetComponentCode      = "meshery-mcp-server-1005"
	ErrListModelsCode        = "meshery-mcp-server-1006"
	ErrGetModelCode          = "meshery-mcp-server-1007"
	ErrListRelationshipsCode = "meshery-mcp-server-1008"

	ErrInvalidIDCode              = "meshery-mcp-server-1023"
	ErrRequiredOneOfCode          = "meshery-mcp-server-1024"
	ErrComponentNotFoundCode      = "meshery-mcp-server-1025"
	ErrModelNotFoundCode          = "meshery-mcp-server-1026"
	ErrInvalidComponentSchemaCode = "meshery-mcp-server-1027"
)

// fromBackend passes a MeshKit error through unchanged, since it already has
// its own cause and remedy, and wraps any other error.
func fromBackend(err error, wrap func(error) error) error {
	if _, ok := errors.Is(err); ok {
		return err
	}
	return wrap(err)
}

func ErrServerRequired() error {
	return errors.New(ErrServerRequiredCode, errors.Alert,
		[]string{"MCP server is required to register tools"},
		[]string{"tools.Register was called with a nil *mcp.Server"},
		[]string{"The server was not constructed before registering tools"},
		[]string{"Construct the MCP server before calling tools.Register"})
}

func ErrBackendRequired() error {
	return errors.New(ErrBackendRequiredCode, errors.Alert,
		[]string{"Meshery backend is required to register tools"},
		[]string{"tools.Register was called with a nil Backend"},
		[]string{"The Meshery backend implementation was not constructed or not passed in"},
		[]string{"Pass a tools.Backend implementation to tools.Register"})
}

// The input schema only checks that a required argument is present, not
// that it is non-empty.
func ErrRequiredArgument(argument string) error {
	return errors.New(ErrRequiredArgumentCode, errors.Alert,
		[]string{"Required tool argument is empty"},
		[]string{fmt.Sprintf("The %q argument must not be empty", argument)},
		[]string{"The tool was called with an empty value for a required argument"},
		[]string{fmt.Sprintf("Call the tool again with a non-empty %q", argument)})
}

func ErrNegativeArgument(argument string, value int) error {
	return errors.New(ErrNegativeArgumentCode, errors.Alert,
		[]string{"Tool argument must not be negative"},
		[]string{fmt.Sprintf("The %q argument is %d", argument, value)},
		[]string{"The tool was called with a negative number"},
		[]string{fmt.Sprintf("Call the tool again with %q set to 0 or more, or leave it out", argument)})
}

func ErrSearchComponents(err error) error {
	return errors.New(ErrSearchComponentsCode, errors.Alert,
		[]string{"Unable to search components"},
		[]string{fmt.Sprintf("Searching components failed: %v", err)},
		[]string{"The Meshery server returned an error or could not be reached"},
		[]string{"Check that the Meshery server is running and reachable, then try again"})
}

func ErrGetComponent(err error, id string) error {
	return errors.New(ErrGetComponentCode, errors.Alert,
		[]string{"Unable to get component"},
		[]string{fmt.Sprintf("Getting component %q failed: %v", id, err)},
		[]string{"The Meshery server returned an error or could not be reached"},
		[]string{"Check that the Meshery server is running and reachable, then try again"})
}

func ErrListModels(err error) error {
	return errors.New(ErrListModelsCode, errors.Alert,
		[]string{"Unable to list models"},
		[]string{fmt.Sprintf("Listing models failed: %v", err)},
		[]string{"The Meshery server returned an error or could not be reached"},
		[]string{"Check that the Meshery server is running and reachable, then try again"})
}

func ErrGetModel(err error, id string) error {
	return errors.New(ErrGetModelCode, errors.Alert,
		[]string{"Unable to get model"},
		[]string{fmt.Sprintf("Getting model %q failed: %v", id, err)},
		[]string{"The Meshery server returned an error or could not be reached"},
		[]string{"Check that the Meshery server is running and reachable, then try again"})
}

func ErrListRelationships(err error) error {
	return errors.New(ErrListRelationshipsCode, errors.Alert,
		[]string{"Unable to list relationships"},
		[]string{fmt.Sprintf("Listing relationships failed: %v", err)},
		[]string{"The Meshery server returned an error or could not be reached"},
		[]string{"Check that the Meshery server is running and reachable, then try again"})
}

func ErrInvalidID(argument, value string) error {
	return errors.New(ErrInvalidIDCode, errors.Alert,
		[]string{"Tool argument is not a valid ID"},
		[]string{fmt.Sprintf("The %q argument %q is not a UUID", argument, value)},
		[]string{"Meshery IDs are UUIDs, such as 5f4e2a1c-8d3b-4c6e-9a7f-1b2c3d4e5f60"},
		[]string{fmt.Sprintf("Call the tool again with %q set to an ID from a previous result", argument)})
}

func ErrRequiredOneOf(arguments ...string) error {
	list := strings.Join(arguments, ", ")
	return errors.New(ErrRequiredOneOfCode, errors.Alert,
		[]string{"A required tool argument is missing"},
		[]string{fmt.Sprintf("At least one of these arguments must be set: %s", list)},
		[]string{"The tool was called without any of the arguments it needs"},
		[]string{fmt.Sprintf("Call the tool again with at least one of: %s", list)})
}

func ErrComponentNotFound(id string) error {
	return errors.New(ErrComponentNotFoundCode, errors.Alert,
		[]string{"Component not found"},
		[]string{fmt.Sprintf("No component has ID %q", id)},
		[]string{"The ID does not belong to a component registered in this Meshery server"},
		[]string{"Find the component with search_components and use the ID it returns"})
}

func ErrModelNotFound(id string) error {
	return errors.New(ErrModelNotFoundCode, errors.Alert,
		[]string{"Model not found"},
		[]string{fmt.Sprintf("No Meshery model has ID %q", id)},
		[]string{"The ID does not belong to a model registered in this Meshery server"},
		[]string{"Find the model with list_models and use the ID it returns"})
}

func ErrInvalidComponentSchema(err error, id string) error {
	return errors.New(ErrInvalidComponentSchemaCode, errors.Alert,
		[]string{"Component schema is not valid JSON"},
		[]string{fmt.Sprintf("Decoding the schema of component %q failed: %v", id, err)},
		[]string{"The component's registrant stored a schema that is not a JSON object"},
		[]string{"Report the component to its model's maintainers; other components are not affected"})
}
