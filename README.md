<div>
    <!-- Top section -->
    <div>
        <img src="https://raw.githubusercontent.com/meshery-extensions/.github/master/profile/assets/img/meshery-extensions-github.png" usemap="#workmap"  />
    </div>

## Meshery MCP Server

The **Meshery MCP Server** exposes Meshery's capabilities to AI assistants through the [Model Context Protocol](https://modelcontextprotocol.io). It runs as a Go binary that speaks MCP over stdio and is being prepared to connect to a running [Meshery Server](https://meshery.io).

> **Status: scaffold.** The project is in its early stages. The Go module, MCP server entrypoint (stdio transport), configuration, and CI are in place. Tooling that integrates with the Meshery REST API is being built incrementally.

### Current scope

- MCP server over stdio, built with the [`mcp-go`](https://github.com/mark3labs/mcp-go) SDK.
- `server_info` tool returning server metadata.
- Environment-driven configuration: `MESHERY_SERVER_URL` (default `http://localhost:9081`) and `MESHERY_API_TOKEN`.

### Getting started

```sh
make build
make run
```

Requires Go 1.26+.

### Contributing

Contributions are welcome. See the [contributing guide](https://github.com/meshery/meshery/blob/master/CONTRIBUTING.md) and the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/main/code-of-conduct.md).

    <!-- Overview section -->
    <div align="center">
        <h3>Meshery is an extensible, self-service engineering plaform for the collaborative management of cloud and cloud native infrastructure.</h3>
        <h3 align="center"><a href="https://meshery.io/extensions">Browse all extensions</a></h3>
        <h5 align="center">
            <a href="https://meshery.io#getting-started">Installation</a> |
            <a href="https://docs.meshery.io">Documentation</a> |
            <a href="https://discuss.meshery.io">Forum</a> |
            <a href="https://play.meshery.io">Playground</a> |
            <a href="https://meshery.io/catalog">Catalog</a>
        </h5>
        <br />
    </div>

Meshery's [high project velocity](https://meshery.io/blog/sixth-highest-velocity-cncf-project) necessitates a revision in its governance and organizational structure to align with the scale of its growing complexity and community contributions. To best serve its expansive ecosystem, Meshery maintainers have opted to partition its numerous GitHub repositories into two distinct organizations: [github.com/meshery](https://github.com/meshery) for the core platform and [meshery-extensions](https://github.com/meshery-extensions) for [extensions](https://meshery.io/extensions) and [integrations](https://meshery.io/integrations).

[Meshery Extensions](https://meshery.io/extension) are plugins or add-ons that enhance the functionality of the Meshery platform beyond its core capabilities. Meshery supports different [types of extensions](https://docs.meshery.io/extensions/)):

- [Academies](https://docs.meshery.io/extensions/academies): Academy extensions enable Meshery as an integrated learning platform.
- [Adapters](https://docs.meshery.io/concepts/architecture/adapters): Adapters allow Meshery to interface with the different cloud native infrastructure.\
- [Build-time](https://docs.meshery.io/reference/extensibility/build-time/): enable integrators to inject custom configurations, data, provider extensions, and other resources directly into the Meshery container image at build-time.
- CLI: Helm and _kubectl_ plugins that let you create Kanvas snapshots from Helm charts, Kubernetes manifests, and the current state of your Kubernetes cluster, then upload them to Meshery.
    - [Kubectl CLI Plugin](https://docs.meshery.io/extensions/kubectl-meshsync-snapshot/)
    - [Helm CLI Plugin](https://docs.meshery.io/extensions/helm-kanvas-snapshot/)
- [Load Generators](https://docs.meshery.io/extensibility/load-generators): for performance characterization and benchmarking.
- [Models](https://docs.meshery.io/extensions/models/): component-based (semantically and non-semantically meaningful) support for a broad variety of platforms, tools, and technologies.
- [Providers](https://docs.meshery.io/extensibility/providers): for connecting to different cloud providers and infrastructure platforms.
- [Schemas](https://docs.meshery.io/reference/extensibility/schemas/) - Meshery schemas are conscientiously extensible via `x-*` vendor extensions.
- [UI Plugins](https://docs.meshery.io/extensibility/ui): Meshery UI has a number of extension points that allow users to customize their experience with third-party plugins.

This organization is managed by Meshery core and extension maintainers. Repositories in this organization need to be sponsored and created by one or more of the core maintainers. Read more about the [rationale for the project's multi-organization approach and it's governance structure](https://meshery.io/blog/2025/meshery-ecosystem-expansion).

<!-- Blog Post and Explanation section -->
<!-- Video Section -->
<h3 align="center">See Meshery and it's plugins in-action</h3>
    <img src="https://raw.githubusercontent.com/meshery/.github/master/profile/assets/img/meshery-dashboard-hero-image.png"  />
<!--     <div align="center"><i>Example extension. See other <a href="https://meshery.io/extensions">Meshery Extensions</a>.<i></div>
    <br /> -->
    <!-- Repositories section -->

# MCP Server

<a href="https://github.com/meshery/meshery/blob/master/GOVERNANCE.md#extensions-githubcommeshery-extensions"><img src="https://img.shields.io/badge/support-community-00B39F?style=flat-square&logo=meshery&logoColor=white"  alt="Level of support for this repo"></a>

<!-- Contributing and Guidelines -->
<div>
    <h2>Community and Contributing</h2>
    <p>Please do! Code and non-code contributions are welcome. This project is community-built and fosters collaboration. Contributors are expected to adhere to the <a href="https://github.com/cncf/foundation/blob/main/code-of-conduct.md"> CNCF Code of Conduct</a>.
    </p>
    <p>Jump into our <a href="https://slack.meshery.io">Slack</a>! Submit your <a href="https://meshery.io/newcomers">community member form</a> access to additional resources. Don't forget to join the <a href="https://meshery.io/calendar">Newcomers meeting</a> held every Thursday!
    </p>
    <img src="https://raw.githubusercontent.com/meshery/meshery/refs/heads/master/.github/assets/images/readme/community.png"
        style="margin:10px;" width="180px" alt="Community" align="right" />
    <ul>
        ✔️ <b>Star</b> ⭐ the main <a href="https://github.com/meshery/meshery">meshery repo</a><br />
        ✔️ <b>Join</b> any or all of the weekly meetings on the <a href="https://meet.meshery.io">community
                calendar</a><br />
        ✔️ <b>Watch</b> <a
                href="https://www.youtube.com/@mesheryio?sub_confirmation=1">community meeting
                recordings</a><br />
        <p>✔️ <b>Access</b> resources by completing a <a href="https://meshery.io/newcomers"> community member form
            </a><br />
        ✔️ <b>Discuss</b> in a Meshery <a href="https://discuss.meshery.io">Community forum</a><br />
        ✔️ Not sure where to start? <b>Grab</b> an open issue with the <a
                href="https://github.com/issues?q=is%3Aopen+is%3Aissue+archived%3Afalse+(org%3Ameshery+OR+org%3Aservice-mesh-performance+OR+org%3Aservice-mesh-patterns+OR+org%3Ameshery-extensions)+label%3A%22help+wanted%22">help-wanted
                label</a><br />
    </ul>
</div>
<!-- Footer Section -->
<img src="https://raw.githubusercontent.com/meshery/.github/master/profile/assets/img/footer.png" align="center" />
