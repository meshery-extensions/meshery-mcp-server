# Meshery MCP Server

<div align="center">
    <img src="https://raw.githubusercontent.com/meshery-extensions/.github/master/profile/assets/img/meshery-extensions-github.png" alt="Meshery Extensions" />
</div>

Meshery MCP Server is an official Meshery extension that provides an interface between MCP-compatible AI clients and Meshery.

The project aims to make Meshery capabilities accessible through the Model Context Protocol (MCP), enabling AI assistants to interact with Meshery and assist with cloud-native design workflows.

## Overview

[Meshery](https://meshery.io/) is a cloud-native engineering platform for collaboratively designing and operating cloud and cloud-native infrastructure.

The Meshery MCP Server extends Meshery with an MCP interface. MCP-compatible clients can communicate with the server using standardized MCP interactions.

This project is being developed in Go and is intended to support MCP-compatible AI clients.

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

> **Note:** The server is currently under development. Installation commands, configuration options, tools, and client integration examples will be documented as the corresponding functionality becomes available.

## Documentation

Project-specific documentation will be maintained in this repository as the MCP server implementation develops.

For Meshery documentation, see:

- [Meshery Documentation](https://docs.meshery.io/)
- [Meshery Quick Start](https://docs.meshery.io/installation/quick-start/)
- [Meshery Extensions](https://docs.meshery.io/extensions/)
- [Meshery REST API Reference](https://docs.meshery.io/reference/rest-apis/)

## Meshery Resources

- [Meshery](https://meshery.io/)
- [Meshery GitHub Repository](https://github.com/meshery/meshery)

## Community and Contributing

Contributions to the Meshery MCP Server are welcome.

Before contributing, please review:

- [Code of Conduct](https://github.com/cncf/foundation/blob/main/code-of-conduct.md)

Join the Meshery community:

- [Community Slack](https://slack.meshery.io/)
- [Community Forum](https://discuss.meshery.io/)
- [Community Calendar](https://meshery.io/calendar)

## License

Meshery MCP Server is licensed under the Apache License 2.0.
