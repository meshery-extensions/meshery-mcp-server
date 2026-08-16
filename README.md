# Meshery MCP Server

<div>
    <img src="https://raw.githubusercontent.com/meshery-extensions/.github/master/profile/assets/img/meshery-extensions-github.png" alt="Meshery MCP Server" />
</div>

Meshery MCP Server is an official Meshery extension that provides an interface between MCP-compatible AI clients and Meshery.

The project aims to make Meshery capabilities accessible through the Model Context Protocol (MCP), enabling AI assistants to interact with Meshery and assist with cloud-native design workflows.

## Overview

[Meshery](https://meshery.io/) is a cloud-native management platform for designing and managing Kubernetes and cloud-native infrastructure.

The Meshery MCP Server extends Meshery with an MCP interface. MCP-compatible clients can communicate with the server using standardized MCP interactions.

This project is being developed in Go and is intended to support AI clients such as Claude Desktop, Cursor, VS Code, and other MCP-compatible applications.

## Architecture

```text
┌─────────────────────────────┐
│         MCP Client          │
│ Claude / Cursor / VS Code   │
│ Cline / other AI clients    │
└──────────────┬──────────────┘
               │
               │ MCP
               ▼
┌─────────────────────────────┐
│      Meshery MCP Server     │
└──────────────┬──────────────┘
               │
               │ Meshery API
               ▼
┌─────────────────────────────┐
│           Meshery           │
└──────────────┬──────────────┘
               │
               ▼
┌─────────────────────────────┐
│ Kubernetes / Cloud-Native   │
│ Infrastructure              │
└─────────────────────────────┘
```

## Project Status

Meshery MCP Server is currently under active development.

The project is being developed to provide:

- MCP server functionality for Meshery.
- Access to Meshery APIs through MCP.
- MCP tools for Meshery capabilities.
- MCP resources for Meshery data.
- MCP prompts for guided workflows.
- Integration with MCP-compatible AI clients.
- Documentation and examples for users and contributors.

> **Note:** The documentation currently available in this repository covers installation, configuration, development, tools, and MCP client integrations. Some advanced MCP capabilities remain under active development and will be documented as they become available.

## Documentation

- [Installation](docs/installation.md)
- [Configuration](docs/configuration.md)
- [Tools Reference](docs/tools-reference.md)
- [Development Guide](docs/development.md)

### AI Client Integrations

- [Claude Desktop](docs/integrations/claude-desktop.md)
- [VS Code / GitHub Copilot](docs/integrations/vscode-copilot.md)
- [Cursor](docs/integrations/cursor.md)
- [Cline](docs/integrations/cline.md)

### Planned Features

The complete MCP tool, resource, prompt, and transport surface is still under development. Documentation for features that are not yet implemented will be added as the corresponding functionality becomes available.

## Meshery Resources

- [Meshery](https://meshery.io/)
- [Meshery Documentation](https://docs.meshery.io/)
- [Meshery GitHub](https://github.com/meshery/meshery)
- [Meshery Extensions](https://meshery.io/extensions)

## Community and Contributing

Contributions to the Meshery MCP Server are welcome.

Before contributing, please review:

- [Contributing Guide](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security Policy](SECURITY.md)

Join the Meshery community:

- [Slack](https://slack.meshery.io/)
- [Community Forum](https://discuss.meshery.io/)
- [Community Calendar](https://meshery.io/calendar)

## License

Meshery MCP Server is licensed under the Apache License 2.0.
