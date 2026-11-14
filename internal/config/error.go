package config

import (
	"fmt"

	"github.com/meshery/meshkit/errors"
)

const (
	ErrReadConfigFileCode            = "replace_me"
	ErrDecodeConfigCode              = "replace_me"
	ErrInvalidMesheryURLCode         = "replace_me"
	ErrInvalidLogLevelCode           = "replace_me"
	ErrInvalidMaxConcurrentCallsCode = "replace_me"
	ErrInvalidMaxRetriesCode         = "replace_me"
)

// ErrReadConfigFile is returned when the --config file cannot be read or parsed.
func ErrReadConfigFile(err error, path string) error {
	return errors.New(ErrReadConfigFileCode, errors.Alert,
		[]string{"Unable to read the config file"},
		[]string{fmt.Sprintf("Reading %q failed: %v", path, err)},
		[]string{"The file passed with --config does not exist, is not readable, or is not valid YAML"},
		[]string{"Check the path given to --config and that the file contains valid YAML"})
}

// ErrDecodeConfig is returned when config values do not match the expected types.
func ErrDecodeConfig(err error) error {
	return errors.New(ErrDecodeConfigCode, errors.Alert,
		[]string{"Unable to decode configuration"},
		[]string{err.Error()},
		[]string{"A setting in the config file has the wrong type"},
		[]string{"Check that every setting in the config file is a plain string"})
}

// ErrInvalidMesheryURL is returned when MESHERY_SERVER_URL / meshery.url is unusable.
// reason must never contain the raw URL, which may embed credentials.
func ErrInvalidMesheryURL(reason string) error {
	return errors.New(ErrInvalidMesheryURLCode, errors.Alert,
		[]string{"Invalid Meshery server URL"},
		[]string{reason},
		[]string{"MESHERY_SERVER_URL or meshery.url is malformed or uses an unsupported scheme"},
		[]string{"Set MESHERY_SERVER_URL to a URL like http://localhost:9081"})
}

// ErrInvalidLogLevel is returned when LOG_LEVEL / log.level is not a supported level.
func ErrInvalidLogLevel(level string) error {
	return errors.New(ErrInvalidLogLevelCode, errors.Alert,
		[]string{"Invalid log level"},
		[]string{fmt.Sprintf("Log level %q is not supported", level)},
		[]string{"LOG_LEVEL or log.level has a typo"},
		[]string{"Set LOG_LEVEL to one of: debug, info, warn, error"})
}

// ErrInvalidMaxConcurrentCalls is returned when meshery.max_concurrent_calls
// / MESHERY_MAX_CONCURRENT_CALLS is less than 1.
func ErrInvalidMaxConcurrentCalls(value int) error {
	return errors.New(ErrInvalidMaxConcurrentCallsCode, errors.Alert,
		[]string{"Invalid max concurrent calls"},
		[]string{fmt.Sprintf("max_concurrent_calls is %d, must be at least 1", value)},
		[]string{"MESHERY_MAX_CONCURRENT_CALLS / meshery.max_concurrent_calls is too small"},
		[]string{"Set it to an integer >= 1"})
}

// ErrInvalidMaxRetries is returned when meshery.max_retries /
// MESHERY_MAX_RETRIES is negative.
func ErrInvalidMaxRetries(value int) error {
	return errors.New(ErrInvalidMaxRetriesCode, errors.Alert,
		[]string{"Invalid max retries"},
		[]string{fmt.Sprintf("max_retries is %d, must not be negative", value)},
		[]string{"MESHERY_MAX_RETRIES / meshery.max_retries is negative"},
		[]string{"Set it to an integer >= 0"})
}
