<div>
    <!-- Top section -->
    <div>
        <img src="https://raw.githubusercontent.com/meshery-extensions/.github/master/profile/assets/img/meshery-extensions-github.png" usemap="#workmap"  />
    </div>
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

## Meshery MCP Server extension

The Meshery MCP Server extension provides a dedicated Model Context Protocol (MCP) server that streamlines and accelerates the creation and management of Meshery designs. As an official Meshery Extension, it is owned end-to-end by its maintainers—covering architecture and design, implementation, tests, documentation, and publication to both the MCP and Skills marketplaces.

The initial scope for the MCP Server, in priority order, includes:

1. Creation of Meshery designs.
2. Retrieval of designs in the user's requested file format (for example, JSON or YAML representations of designs).
3. Snapshotting of Meshery designs.
4. Design deployment dry-run results.
5. Definition, invocation, and results of performance tests.

Contributors who are capable of and available to carry this work through independently—from 0% to 100%—may become the project's official maintainer(s).

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
<div>
    <h2>Repositories</h2>
    <table border="0px" align="center">
        <tr>
            <!-- kanvas-site -->
            <td style="padding: 28px;">
                <h2 align="left"><a href="https://github.com/meshery-extensions/kanvas-site">kanvas-site</a></h2>
                <img src="https://raw.githubusercontent.com/meshery-extensions/.github/master/profile/assets/img/meshery-extensions-color.svg" style="margin-right:10px;" width="75px" alt="Meshery Logo" align="left" />
                <p>The documentation, platform web application site, and landing interface for Kanvas—the collaborative visual designer for cloud-native infrastructure.</p>
                <p align="left">
                    <a href="https://github.com/meshery-extensions/kanvas-site/graphs/contributors"><img src="https://img.shields.io/github/contributors/meshery-extensions/kanvas-site.svg" /></a>
                    <a href="https://github.com/issues?q=is%3Aopen+is%3Aissue+archived%3Afalse+org%3Ameshery-extensions+repo%3Akanvas-site+label%3A%22help+wanted%22+"><img src="https://img.shields.io/github/issues/meshery-extensions/kanvas-site/help%20wanted.svg?color=informational" /></a>
                </p>
            </td>
        </tr>
        <tr>
            <!-- kanvas-snapshot -->
            <td style="padding: 28px;">
                <h2 align="left"><a href="https://github.com/meshery-extensions/kanvas-snapshot">kanvas-snapshot</a></h2>
                <img src="https://raw.githubusercontent.com/meshery-extensions/.github/master/profile/assets/img/meshery-extensions-color.svg" style="margin-right:10px;" width="75px" alt="Meshery Logo" align="left" />
                <p>The core rendering library and shared framework driving visual topology captures, state comparison engines, and design export workflows across CLI and Helm extensions.</p>
                <p align="left">
                    <a href="https://github.com/meshery-extensions/kanvas-snapshot/graphs/contributors"><img src="https://img.shields.io/github/contributors/meshery-extensions/kanvas-snapshot.svg" /></a>
                    <a href="https://github.com/issues?q=is%3Aopen+is%3Aissue+archived%3Afalse+org%3Ameshery-extensions+repo%3Akanvas-snapshot+label%3A%22help+wanted%22+"><img src="https://img.shields.io/github/issues/meshery-extensions/kanvas-snapshot/help%20wanted.svg?color=informational" /></a>
                </p>
            </td>
        </tr>
        <tr>
            <!-- kubectl-kanvas-snapshot -->
            <td style="padding: 28px;">
                <h2 align="left"><a href="https://github.com/meshery-extensions/kubectl-kanvas-snapshot">kubectl-kanvas-snapshot</a></h2>
                <img src="https://raw.githubusercontent.com/meshery-extensions/.github/master/profile/assets/img/meshery-extensions-color.svg" style="margin-right:10px;" width="75px" alt="Meshery Logo" align="left" />
                <p>A native command-line <code>kubectl</code> plugin configured to easily generate exportable architectural design blueprints and snapshots of live Kubernetes clusters.</p>
                <p align="left">
                    <a href="https://github.com/meshery-extensions/kubectl-kanvas-snapshot/graphs/contributors"><img src="https://img.shields.io/github/contributors/meshery-extensions/kubectl-kanvas-snapshot.svg" /></a>
                    <a href="https://github.com/issues?q=is%3Aopen+is%3Aissue+archived%3Afalse+org%3Ameshery-extensions+repo%3Akubectl-kanvas-snapshot+label%3A%22help+wanted%22+"><img src="https://img.shields.io/github/issues/meshery-extensions/kubectl-kanvas-snapshot/help%20wanted.svg?color=informational" /></a>
                </p>
            </td>
        </tr>
        <tr>
            <!-- helm-kanvas-snapshot -->
            <td style="padding: 28px;">
                <h2 align="left"><a href="https://github.com/meshery-extensions/helm-kanvas-snapshot">helm-kanvas-snapshot</a></h2>
                <img src="https://raw.githubusercontent.com/meshery-extensions/.github/master/profile/assets/img/meshery-extensions-color.svg" style="margin-right:10px;" width="75px" alt="Meshery Logo" align="left" />
                <p>A Helm plugin extension engineered to generate exportable visual architectural maps and structural snapshots directly from packaged Helm charts.</p>
                <p align="left">
                    <a href="https://github.com/meshery-extensions/helm-kanvas-snapshot/graphs/contributors"><img src="https://img.shields.io/github/contributors/meshery-extensions/helm-kanvas-snapshot.svg" /></a>
                    <a href="https://github.com/issues?q=is%3Aopen+is%3Aissue+archived%3Afalse+org%3Ameshery-extensions+repo%3Ahelm-kanvas-snapshot+label%3A%22help+wanted%22+"><img src="https://img.shields.io/github/issues/meshery-extensions/helm-kanvas-snapshot/help%20wanted.svg?color=informational" /></a>
                </p>
            </td>
        </tr>
        <tr>
            <!-- meshery-nighthawk -->
            <td style="padding: 28px;">
                <h2 align="left"><a href="https://github.com/meshery-extensions/meshery-nighthawk">meshery-nighthawk</a></h2>
                <img src="https://raw.githubusercontent.com/meshery-extensions/.github/master/profile/assets/img/meshery-extensions-color.svg" style="margin-right:10px;" width="75px" alt="Meshery Logo" align="left" />
                <p>An extension adapter designed for managing and orchestrating Nighthawk—the distributed layer-seven traffic and performance benchmarking subsystem.</p>
                <p align="left">
                    <a href="https://github.com/meshery-extensions/meshery-nighthawk/graphs/contributors"><img src="https://img.shields.io/github/contributors/meshery-extensions/meshery-nighthawk.svg" /></a>
                    <a href="https://github.com/issues?q=is%3Aopen+is%3Aissue+archived%3Afalse+org%3Ameshery-extensions+repo%3Ameshery-nighthawk+label%3A%22help+wanted%22+"><img src="https://img.shields.io/github/issues/meshery-extensions/meshery-nighthawk/help%20wanted.svg?color=informational" /></a>
                </p>
            </td>
        </tr>
        <tr>
            <!-- meshery-extensions-packages -->
            <td style="padding: 28px;">
                <h2 align="left"><a href="https://github.com/meshery-extensions/meshery-extensions-packages">meshery-extensions-packages</a></h2>
                <img src="https://raw.githubusercontent.com/meshery-extensions/.github/master/profile/assets/img/meshery-extensions-color.svg" style="margin-right:10px;" width="75px" alt="Meshery Logo" align="left" />
                <p>The centralized distribution hub for packaging, versioning, and releasing bundled assets, plugins, and compiled components across the extended Meshery ecosystem.</p>
                <p align="left">
                    <a href="https://github.com/meshery-extensions/meshery-extensions-packages/graphs/contributors"><img src="https://img.shields.io/github/contributors/meshery-extensions/meshery-extensions-packages.svg" /></a>
                    <a href="https://github.com/issues?q=is%3Aopen+is%3Aissue+archived%3Afalse+org%3Ameshery-extensions+repo%3Ameshery-extensions-packages+label%3A%22help+wanted%22+"><img src="https://img.shields.io/github/issues/meshery-extensions/meshery-extensions-packages/help%20wanted.svg?color=informational" /></a>
                </p>
            </td>
        </tr>
        <tr>
            <!-- meshery-academy -->
            <td style="padding: 28px;">
                <h2 align="left"><a href="https://github.com/meshery-extensions/meshery-academy">meshery-academy</a></h2>
                <img src="https://raw.githubusercontent.com/meshery-extensions/.github/master/profile/assets/img/meshery-extensions-color.svg" style="margin-right:10px;" width="75px" alt="Meshery Logo" align="left" />
                <p>The central community interactive learning repository hosting courseware, lab setups, and certification trees for Meshery's broad extension ecosystem.</p>
                <p align="left">
                    <a href="https://github.com/meshery-extensions/meshery-academy/graphs/contributors"><img src="https://img.shields.io/github/contributors/meshery-extensions/meshery-academy.svg" /></a>
                    <a href="https://github.com/issues?q=is%3Aopen+is%3Aissue+archived%3Afalse+org%3Ameshery-extensions+repo%3Ameshery-academy+label%3A%22help+wanted%22+"><img src="https://img.shields.io/github/issues/meshery-extensions/meshery-academy/help%20wanted.svg?color=informational" /></a>
                </p>
            </td>
        </tr>
    </table>
</div>
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
</div>
