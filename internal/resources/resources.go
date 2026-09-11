package resources

import (
	"errors"
	"fmt"
)

var ErrInvalidURI = errors.New("invalid resource URI")

// ListResources returns all supported MCP resource URIs
func ListResources() []string {
	return []string{
		"meshery://connections",
		"meshery://providers",
		"meshery://adapters",
		"meshery://health",
		"meshery://environments",
	}
}

// ReadResource routes resource requests to the correct handler
func ReadResource(uri string) (interface{}, error) {
	switch uri {

	case "meshery://connections":
		return GetConnections()

	case "meshery://providers":
		return GetProviders()

	case "meshery://adapters":
		return GetAdapters()

	case "meshery://health":
		return GetHealth()

	case "meshery://environments":
		return GetEnvironments()

	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidURI, uri)
	}
}
