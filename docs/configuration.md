# Configuration

Meshery MCP Server supports configuration through a YAML configuration file and environment variables.

## Configuration File

The default configuration file is:

    ~/.meshery/mcp-config.yaml

The configuration file can contain multiple Meshery contexts.

### Configuration Format

    current-context: dev

    contexts:
      dev:
        server: http://localhost:9081
        token: <your-meshery-api-token>
        provider: Meshery

      production:
        server: https://meshery.example.com
        token: <your-meshery-api-token>
        provider: Meshery

### Configuration Fields

| Field | Description |
| --- | --- |
| `current-context` | Name of the currently active context |
| `contexts` | Map containing the configured Meshery contexts |
| `server` | Base URL of the Meshery Server REST API |
| `token` | API token used for authentication |
| `provider` | Meshery provider name, such as `Meshery` or `None` |

The active context must exist in the `contexts` map and must contain a valid server URL.

## Custom Configuration File

To use a configuration file at a different location, set:

    MESHERY_CONFIG_PATH

Example:

    export MESHERY_CONFIG_PATH=/path/to/mcp-config.yaml

On PowerShell:

    $env:MESHERY_CONFIG_PATH="C:\path\to\mcp-config.yaml"

## Environment Variables

Environment variables can be used instead of a configuration file or to override values in the active context.

### MESHERY_SERVER_URL

Specifies the Meshery server URL.

Default:

    http://localhost:9081

Example:

    export MESHERY_SERVER_URL=http://localhost:9081

PowerShell:

    $env:MESHERY_SERVER_URL="http://localhost:9081"

### MESHERY_API_TOKEN

Specifies the Meshery API token used for authentication.

Example:

    export MESHERY_API_TOKEN=<your-meshery-api-token>

PowerShell:

    $env:MESHERY_API_TOKEN="<your-meshery-api-token>"

Do not commit API tokens or other credentials to Git.

### MESHERY_PROVIDER

Specifies the Meshery provider associated with the active context.

Example:

    export MESHERY_PROVIDER=Meshery

PowerShell:

    $env:MESHERY_PROVIDER="Meshery"

### MESHERY_CONTEXT

Specifies which configured context should be active.

Example:

    export MESHERY_CONTEXT=production

PowerShell:

    $env:MESHERY_CONTEXT="production"

If the specified context does not exist, the existing configured context selection is not replaced.

## Configuration Priority

When the configuration is loaded, Meshery MCP Server:

1. Attempts to load the configuration file.
2. If the configuration file does not exist, creates a configuration from environment variables.
3. Applies environment variable overrides to the active context.

Environment variables override configuration-file values for the active context.

The following environment variables can override configuration:

- `MESHERY_CONTEXT`
- `MESHERY_SERVER_URL`
- `MESHERY_API_TOKEN`
- `MESHERY_PROVIDER`

## Using Multiple Meshery Contexts

Multiple Meshery instances can be configured in the same file.

Example:

    current-context: dev

    contexts:
      dev:
        server: http://localhost:9081
        token: <development-token>
        provider: Meshery

      staging:
        server: https://staging.example.com
        token: <staging-token>
        provider: Meshery

      production:
        server: https://meshery.example.com
        token: <production-token>
        provider: Meshery

The active context is selected using `current-context`.

You can also select a context using:

    export MESHERY_CONTEXT=staging

## Configuration Validation

The configuration is validated when loaded.

The active context must:

- Exist in the `contexts` map.
- Have a non-empty `server` value.
- Have a valid server URL.

The configuration parser also rejects unknown YAML fields.

For example, a configuration using an unsupported field such as:

    contexts:
      dev:
        endpoint: http://localhost:9081

is rejected.

Use `server` instead:

    contexts:
      dev:
        server: http://localhost:9081

## Authentication

Authenticated Meshery requests use the configured API token and provider.

Example:

    contexts:
      dev:
        server: http://localhost:9081
        token: <your-meshery-api-token>
        provider: Meshery

Keep credentials private and never commit real tokens to source control.

## Environment-Only Configuration

For a quick local setup without a configuration file, set the environment variables:

    export MESHERY_SERVER_URL=http://localhost:9081
    export MESHERY_API_TOKEN=<your-meshery-api-token>
    export MESHERY_PROVIDER=Meshery

On PowerShell:

    $env:MESHERY_SERVER_URL="http://localhost:9081"
    $env:MESHERY_API_TOKEN="<your-meshery-api-token>"
    $env:MESHERY_PROVIDER="Meshery"

If no configuration file is found, these environment variables are used to create the default context.

## Security

- Never commit API tokens to source control.
- Do not place real credentials in documentation examples.
- Use environment variables or a local configuration file for development credentials.
- Protect configuration files containing authentication tokens.

## Related Documentation

- [Installation](installation.md)
- [Tools Reference](tools-reference.md)
- [Development Guide](development.md)
