# Installation

> **Status:** Meshery MCP Server is currently under active development.

Meshery MCP Server is an official Meshery extension that provides an MCP interface for interacting with Meshery from MCP-compatible clients.

## Prerequisites

Before working with Meshery MCP Server, you should have:

- A running Meshery instance.
- An MCP-compatible client.
- Git installed.
- Go installed if you plan to build the server from source.

For information about installing Meshery, see the [Meshery Quick Start](https://docs.meshery.io/installation/quick-start/).

## Clone the Repository

Clone the official Meshery MCP Server repository:

    git clone https://github.com/meshery-extensions/meshery-mcp-server.git
    cd meshery-mcp-server

## Build from Source

The Meshery MCP Server is currently under active development.

The repository does not yet provide a finalized Go module, build target, or release binary. Therefore, the exact build and run commands will be documented here once the implementation is available.

The expected source-build workflow is:

    git clone https://github.com/meshery-extensions/meshery-mcp-server.git
    cd meshery-mcp-server

    # Build and run commands will be added
    # when the server implementation is finalized.

## Pre-built Binaries

Official pre-built binaries will be documented here when Meshery MCP Server releases are published.

The release documentation will include:

- Supported operating systems.
- Supported CPU architectures.
- Download locations.
- Installation instructions.
- Version verification.

## Docker

Docker installation will be documented once an official Meshery MCP Server container image is published.

The documentation will include:

- Official container image name.
- Available image tags.
- Required environment variables.
- Port configuration.
- Example Docker commands.

## Homebrew

Homebrew installation instructions will be added when an official Homebrew distribution becomes available.

## Verify the Installation

Once the server implementation and command-line interface are finalized, this section will document how to verify the installation.

Verification should confirm that:

1. Meshery MCP Server starts successfully.
2. The server can communicate with Meshery.
3. An MCP-compatible client can connect to the server.
4. MCP tools and resources can be discovered.

## Next Steps

After installing Meshery MCP Server:

1. Configure the Meshery server URL.
2. Configure authentication if required.
3. Configure your MCP-compatible client.
4. Start the MCP server.
5. Verify the connection.
6. Use the available MCP tools and resources.
