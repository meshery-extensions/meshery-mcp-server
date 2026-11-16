package meshery

import (
	"fmt"

	"github.com/meshery/meshkit/errors"
)

const (
	ErrNotImplementedCode = "meshery-mcp-server-1009"
)

func ErrNotImplemented(operation string) error {
	return errors.New(ErrNotImplementedCode, errors.Alert,
		[]string{"Not implemented"},
		[]string{fmt.Sprintf("%s does not call the Meshery server yet", operation)},
		[]string{"This tool is a placeholder until its Meshery API call is implemented"},
		[]string{"Do not rely on this tool; it always returns this error for now"})
}
