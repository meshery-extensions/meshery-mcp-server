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
| `contexts` | Map containing configured Meshery contexts |
| `server` | Base URL of the Meshery Server REST API |
| `token` | API token used for authentication |
| `provider` | Meshery provider name |

The active context must exist in `contexts` and must contain a valid server URL.

## Custom Configuration File

Set `MESHERY_CONFIG_PATH` to use a different configuration file:

    export MESHERY_CONFIG_PATH=/path/to/mcp-config.yaml

PowerShell:

    $env:MESHERY_CONFIG_PATH="C:\path	o\mcp-config.yaml"

## Environment Variables

Environment variables can be used instead of a configuration file or to override values from the active context.

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

Replace `<your-meshery-api-token>` with your real Meshery API token. Do not include a real token in documentation or source control.

Example:

    export MESHERY_API_TOKEN="<your-meshery-api-token>"

PowerShell:

    $env:MESHERY_API_TOKEN="<your-meshery-api-token>"

### MESHERY_PROVIDER

Specifies the Meshery provider associated with the active context.

Example:

    export MESHERY_PROVIDER=Meshery

PowerShell:

    $env:MESHERY_PROVIDER="Meshery"

### MESHERY_CONTEXT

Specifies the configured context to use.

Example:

    export MESHERY_CONTEXT=production

PowerShell:

    $env:MESHERY_CONTEXT="production"

If the specified context does not exist, the configured context selection is not replaced.

## Configuration Priority

When configuration is loaded:

1. The configuration file is loaded when available.
2. If the configuration file is unavailable, environment variables can provide the configuration.
3. Environment variables override values for the active context.

The following variables can override configuration:

- `MESHERY_CONTEXT`
- `MESHERY_SERVER_URL`
- `MESHERY_API_TOKEN`
- `MESHERY_PROVIDER`

## Multiple Meshery Contexts

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

You can override it with:

    export MESHERY_CONTEXT=staging

PowerShell:

    $env:MESHERY_CONTEXT="staging"

## Validation

The configuration is validated when loaded.

The active context must:

- Exist in the `contexts` map.
- Have a non-empty `server` value.
- Have a valid server URL.

Unknown YAML fields are rejected.

## Authentication

Authenticated Meshery requests use the configured API token and provider.

Keep credentials private and never commit real tokens.

## Environment-Only Configuration

For a local setup without a configuration file:

    export MESHERY_SERVER_URL=http://localhost:9081
    export MESHERY_API_TOKEN="<your-meshery-api-token>"
    export MESHERY_PROVIDER=Meshery

PowerShell:

    $env:MESHERY_SERVER_URL="http://localhost:9081"
    $env:MESHERY_API_TOKEN="<your-meshery-api-token>"
    $env:MESHERY_PROVIDER="Meshery"

Replace the token placeholder with your real Meshery API token before using the configuration.

## Security

- Never commit API tokens to source control.
- Never put real credentials in documentation examples.
- Use environment variables or a local configuration file for development credentials.
- Protect configuration files containing authentication tokens.

## Related Documentation

- [Installation](installation.md)
- [Tools Reference](tools-reference.md)
- [Development Guide](development.md)
