// Package config loads Meshery server connection settings from the process
// environment.
package config

import "os"

// DefaultMeshServerURL is used when MESHERY_SERVER_URL is not set.
const DefaultMeshServerURL = "http://localhost:9081"

// DefaultMeshProvider is used when MESHERY_PROVIDER is not set.
const DefaultMeshProvider = "Meshery"

// Config holds the settings needed to talk to a Meshery Server instance.
type Config struct {
	MeshServerURL string
	MeshAPIToken  string
	MeshProvider  string
}

// Load reads configuration from environment variables, applying defaults for
// anything unset.
func Load() *Config {
	return &Config{
		MeshServerURL: envOr("MESHERY_SERVER_URL", DefaultMeshServerURL),
		MeshAPIToken:  os.Getenv("MESHERY_API_TOKEN"),
		MeshProvider:  envOr("MESHERY_PROVIDER", DefaultMeshProvider),
	}
}

// envOr returns the environment variable's value, or fallback if it is unset.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
