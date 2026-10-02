# AGENTS.md — OVN-Kubernetes

Project context for AI coding agents. See [agents.md](https://agents.md/) for the open standard.

## Start Here

This file applies repository-wide. Read the additional `AGENTS.md` files along
the path to code you change; their more specific guidance applies within that
subtree:

- [`go-controller/AGENTS.md`](go-controller/AGENTS.md): binaries, component
  architecture, Go conventions, and generators.
- [`go-controller/pkg/AGENTS.md`](go-controller/pkg/AGENTS.md): package map,
  network topologies, OVN database rules, and test helpers.
- [`go-controller/pkg/crd/AGENTS.md`](go-controller/pkg/crd/AGENTS.md): API
  compatibility, validation, generated artifacts, and admission tests.

Before editing, inspect `git status --short` and preserve existing local changes.
Read the affected implementation and nearby tests, and keep changes scoped to
the task. Use the Makefiles, scripts, and CI workflows in this checkout as the
source of truth for commands and tool versions.

## Project Overview

OVN-Kubernetes is a network plugin written according to CNI Spec that provides
networking for Kubernetes clusters with Open Virtual Network (OVN) and
Open vSwitch (OVS) at its core.

| Feature | Description |
|---------|-------------|
| Pod Networking | Pod-to-pod connectivity via OVN logical switches and routers |
| IPAM | IP address management for pods and networks |
| Services | Kubernetes Services implemented as OVN load balancers |
| Endpoint Slices | Scalable endpoint tracking for services |
| NetworkPolicy | Kubernetes NetworkPolicy enforcement via OVN ACLs |
| AdminNetworkPolicy | Cluster-scoped network policy (ANP/BANP) |
| EgressIP | Source IP control for egress traffic |
| EgressFirewall | Egress traffic filtering rules |
| EgressService | Egress traffic routing through services |
| EgressQoS | QoS marking on egress traffic |
| Multi-Egress Gateway | Egress traffic via multiple gateway nodes |
| User Defined Networks | Multi-network support, network segmentation (UDN) |
| Cluster Network Connect | Connecting isolated User Defined Networks together for controlled inter-UDN connectivity |
| Multi-Homing | Pods attached to multiple networks |
| DPU/SmartNIC Offload | Hardware acceleration via OVS offload |
| Multicast | IGMP snooping and relay via OVN |
| NetworkQoS | DSCP marking and traffic shaping |
| BGP | Route advertisements and peering |
| EVPN | Ethernet VPN integration |
| No-Overlay | Direct pod routing using BGP-learned routes, without encapsulation |
| KubeVirt | VM live migration and persistent IPs support |
| Hybrid Overlay | Mixed Windows/Linux clusters via VXLAN |

## Repository Layout

```text
go-controller/        # Main Go codebase — feature implementations, component code, ovnkube binaries, CRDs, libovsdb models, observability library, hybrid-overlay
test/e2e/             # End-to-end Ginkgo tests covering all features (network policy, egress, UDN, KubeVirt, BGP, etc.)
test/crd-integration/ # CRD integration tests (defaulting, validation, CEL) — run via envtest, no Kind cluster needed
dist/                 # Container images (Dockerfiles) and deployment YAML manifests
helm/                 # Helm charts for deploying ovn-kubernetes
docs/                 # mkdocs source for ovn-kubernetes.io — OKEPs, feature docs, design docs, developer, installation guides
contrib/              # Kind cluster scripts, local deployment helpers (kind.sh), and development tooling
hack/                 # Repository-wide tooling, including documentation builds
.github/workflows/    # CI checks and supported test configurations
LICENSES/             # License and third-party package licensing information
```

The repository root is not a Go module. The main module is `go-controller/`;
`test/e2e/`, `test/crd-integration/`, and `test/conformance/` have their own
`go.mod` files. Run Go commands from the module you are changing. Scope searches
to relevant directories and exclude `go-controller/vendor/` unless investigating
a dependency.

## Build and Test

Use the Go version specified by the relevant `go.mod` (`go` and `toolchain`
directives). The main module builds and tests with vendored dependencies.
Run these commands from `go-controller/`:

```bash
cd go-controller/
make build          # Build binaries into _output/go/bin/
make lint           # Check formatting and lint rules (required for Go PRs)
make test           # Run unit tests with the race detector
```

`make lint-fix` applies automatic fixes; inspect its diff afterward. Lint and
test targets use Podman/Docker when available and otherwise run natively.
Some unit tests manipulate network namespaces and need Linux networking
capabilities; the test target uses a privileged container or invokes `sudo`
for the packages that require it. Native race tests need a C compiler.

During development, run the affected packages first. Examples from
`go-controller/` (replace the package and focus expression as needed):

```bash
make test PKGS=./pkg/ovn
# Native execution of a focused Ginkgo test through the repository wrapper:
RACE=1 PKGS=./pkg/ovn hack/test-go.sh focus 'test description'
```

Prefer the wrapper over bare `go test` for controller tests: it sets
`KUBE_FEATURE_WatchListClient=false` for fake-client compatibility, manages
package-specific timeouts, and handles privileged tests. `-run` selects Go test
functions; use Ginkgo focus to select individual specs within a Ginkgo suite.

Choose coverage that exercises the behavior being changed:

| Change | Test location and validation |
|--------|------------------------------|
| Go logic or controller behavior | Nearby `*_test.go` files; use existing package test helpers and add regression coverage for bug fixes |
| CRD fields, defaulting, or validation | `test/crd-integration/`; run `make -C test test-crd` from the repository root |
| Networking behavior or new features | `test/e2e/`; add or update end-to-end coverage and run the relevant Kind tests |
| Website content or navigation | Run `mkdocs build --strict` from the root with the dependencies used by `.github/workflows/docs.yml` |
| Agent instructions or other Markdown outside the website | Check local links, command accuracy, and `git diff --check` |

CRD admission tests (defaulting, validation, CEL rules) belong in `test/crd-integration/`,
not in `test/e2e/`. They use envtest to start a real API server and etcd locally;
no Kind cluster is needed. The Make target downloads the required binaries on
first use. See the [CRD integration test guide](docs/developer-guide/local_testing_guide.md#crd-integration-tests).

E2E tests require a configured Kind cluster, feature flags, and test environment.
Follow the [local testing guide](docs/developer-guide/local_testing_guide.md) and
the relevant lane in [CI](.github/workflows/test.yml). After setup, run an
OVN-Kubernetes test with `make -C test control-plane WHAT='test description'`
from the root.

Before delivering Go changes, run the build, lint, and unit-test checks above,
plus checks required by the affected API or feature. For documentation-only
changes, use the relevant documentation checks. Report the commands run and
their results, and identify any checks blocked by missing tools or environment.

## Generated Files and Dependencies

Edit generator inputs and regenerate outputs; do not hand-edit generated Go,
CRD manifests, database models, or mocks. Run these targets from `go-controller/`
when their inputs change:

| Input changed | Command | Outputs to review |
|---------------|---------|-------------------|
| CRD Go types or validation markers | `make codegen` | Generated Go under `pkg/crd/` and CRD YAML in `helm/ovn-kubernetes/crds/` |
| OVN/OVS schemas or model generator inputs | `make modelgen` | Generated models in `pkg/nbdb/`, `pkg/sbdb/`, and `pkg/vswitchd/` |
| Mocked interfaces or `.mockery.yaml` | `make mocksgen` | Mock files at the locations configured in `.mockery.yaml` |

Include the corresponding generated changes. CI checks code and mocks with
`make verify-codegen` and `make verify-mocksgen`. See the
[generator guide](docs/developer-guide/developer.md) for setup and details.
CRD changes also require API reference updates (see [Documentation](#documentation)).

Keep dependency changes intentional: do not edit `vendor/` directly or update
module files as an incidental fix. When changing dependencies, keep `go.mod`,
`go.sum`, vendored code, and third-party licenses consistent; use the
`verify-go-mod-vendor`, `third-party-licenses`, and `verify-third-party-licenses`
targets in `go-controller/Makefile` as applicable.

## Key Conventions

Follow existing package patterns. For new controllers, read the
[level-driven controller guidelines](docs/developer-guide/developer.md#level-driven-controllers):
use the shared controller framework and network manager, and design for multiple
networks. Consider IPv4, IPv6, gateway modes, and UDN isolation when changing
network behavior.

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for full details:

- [Commit message guidelines](CONTRIBUTING.md#commit-message-guidelines)
- [Pull request checklist](CONTRIBUTING.md#pull-request-checklist)
- [Sign your commits (DCO)](CONTRIBUTING.md#sign-your-commits)
- [AI guidelines](CONTRIBUTING.md#ai-guidelines)

When preparing commits, use the component-prefixed message format and DCO
sign-off. When preparing a PR, disclose material AI assistance in its
"Special notes for your reviewer" section as required by the contribution guide.

## OKEPs (Enhancement Proposals)

New features require an OKEP in `docs/okeps/`. See `docs/okeps/okep-4368-template.md` for the
template. OKEPs must have an associated GitHub issue, cover all template sections and update
`mkdocs.yml`.

## Architecture Notes

The OVN-Kubernetes plugin watches the Kubernetes API. It acts on the generated Kubernetes
cluster events by creating and configuring the corresponding OVN logical constructs in the
OVN database for those events. OVN (which is an abstraction on top of Open vSwitch) converts
these logical constructs into logical flows in its database and programs the OpenFlow flows
on the node, which enables networking on a Kubernetes cluster.

See [Architecture](docs/design/architecture.md) for component details (ovnkube-control-plane,
ovnkube-node, ovs-node pods and their containers).

### Gateway Modes

OVN-Kubernetes supports two gateway modes that affect how traffic enters and
leaves the cluster:
- **Local gateway (lgw)** — Traffic leaving the pod to go outside the cluster leaves the OVN stack
  and enters the host networking stack via the management port (mp-X) and then based on routes on
  the host, it either leaves via breth0/primary node NIC or via other interfaces on the host. This
  mode uses nftables to implement a lot of the service traffic and since traffic leaves the OVN/OVS
  datapath, it is not hardware offloadable.
- **Shared gateway (sgw)** — Traffic leaving the pod to go outside the cluster stays within the
  OVN/OVS datapath. It goes from the pod through the OVN logical switch, to the Gateway Router
  (GR), and out via the OVS bridge (breth0) directly. This mode is hardware offloadable to DPUs/SmartNICs.

To determine which mode a cluster is using, check the `k8s.ovn.org/l3-gateway-config`
annotation on any node — the `mode` field will be `"local"` or `"shared"`.
Shared gateway is the default.

## Documentation

Update feature or design documentation when user-facing behavior or architecture
changes. Use relative Markdown links between documentation pages and register
new pages in `mkdocs.yml`. See the
[documentation guide](docs/developer-guide/documentation.md).

For new CRDs and CRD type changes, regenerate the API reference from the
repository root:

```bash
make -C docs generate-api-reference
```

For a new CRD, also update the generator's CRD mapping in `docs/Makefile`,
`docs/api-reference/introduction.md`, and the API Reference navigation in
`mkdocs.yml`.
