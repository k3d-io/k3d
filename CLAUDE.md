# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

k3d creates containerized k3s (lightweight Kubernetes) clusters using Docker. It runs multi-node k3s clusters on a single machine. The Go module is `github.com/k3d-io/k3d/v5`.

## Build & Development Commands

```bash
make build              # Build for local platform -> bin/k3d
make build-debug        # Build with debug symbols
make build-cross        # Cross-compile for all platforms (needs gox)
make install-tools      # Install required tools (golangci-lint, gox)
```

Dependencies are vendored (`-mod=vendor`). The go.work workspace includes `.`, `./docgen`, and `./tools`.

### Local Helper Image Overrides

When developing helper images locally, use these env vars to point k3d at your local builds instead of the published ones (see `pkg/types/env.go`):

```bash
K3D_IMAGE_LOADBALANCER=ghcr.io/k3d-io/k3d-proxy:dev   # Override the load balancer (proxy) image
K3D_IMAGE_TOOLS=ghcr.io/k3d-io/k3d-tools:dev           # Override the tools helper image
```

## Testing

```bash
make test                                           # Run all unit tests
go test ./pkg/config/... -run TestProcessConfig     # Run a specific test
make e2e                                            # Run E2E tests (requires Docker, uses DIND)
make e2e -e E2E_INCLUDE="test_basic"                # Run specific E2E test
make e2e -e E2E_LOG_LEVEL=trace -e E2E_FAIL_FAST=true  # E2E with debug output
```

E2E tests are bash scripts in `tests/` that run inside Docker-in-Docker containers. Unit tests use `testify` for assertions and `go-test/deep` for deep equality. Config test assets are in `pkg/config/test_assets/`.

### E2E Test Framework

**Execution chain:** `make e2e` -> `tests/dind.sh` (builds DIND image, starts privileged container) -> `tests/runner.sh` (discovers and batches tests) -> `test_*.sh` scripts.

**Writing a new E2E test:**

1. Create `tests/test_<name>.sh` (the `test_` prefix is required for auto-discovery)
2. Make it executable (`chmod +x`)
3. Follow the standard boilerplate:
   ```bash
   #!/bin/bash
   CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
   [ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)"; exit 1; }
   source "$CURR_DIR/common.sh"

   ### Step Setup ###
   LOG_FILE="$TEST_OUTPUT_DIR/$( basename "${BASH_SOURCE[0]}" ).log"
   exec >${LOG_FILE} 2>&1
   export LOG_FILE
   KUBECONFIG="$KUBECONFIG_ROOT/$( basename "${BASH_SOURCE[0]}" ).yaml"
   export KUBECONFIG
   ### Step Setup ###

   export CURRENT_STAGE="Test | <name>"
   # ... test logic using $EXE, info, failed, check_clusters, etc.
   exit 0
   ```
4. Use unique cluster/resource names to avoid collisions with parallel tests
5. Always clean up resources (cluster delete, registry delete) at the end

**Key helpers from `tests/common.sh`:** `info`, `failed`, `passed`, `highlight`, `check_clusters`, `check_multi_node`, `check_registry`, `wait_for_pod_running_by_name`, `wait_for_pod_running_by_label`, `exec_in_node`, `docker_assert_container_label`, `k3s_assert_node_label`.

**E2E environment variables:** `E2E_INCLUDE` (run only named tests), `E2E_EXCLUDE` (skip named tests), `E2E_PARALLEL` (default: 4), `E2E_EXTRA` (run `extra_test_*` files), `E2E_FAIL_FAST`, `E2E_LOG_LEVEL`, `E2E_K3S_VERSION`, `E2E_DIND_VERSION`, `E2E_KEEP` (keep runner container).

## Linting & Formatting

```bash
make fmt        # Fix formatting (gofmt)
make check-fmt  # Check formatting
make lint       # Run golangci-lint (v2.4.0)
make ci-lint    # Same with 5-minute timeout (CI)
make check      # check-fmt + lint
```

## Architecture

Three-layer design:

1. **CLI layer** (`cmd/`) - Cobra commands for cluster, node, registry, kubeconfig, image, config, debug subcommands. Entry point: `cmd/root.go` -> `NewCmdK3d()`.

2. **Client layer** (`pkg/client/`) - Business logic and orchestration. Key files: `cluster.go` (ClusterCreate/Start/Delete/Run), `node.go` (NodeCreate/Start/Delete), `registry.go`, `loadbalancer.go`, `kubeconfig.go`.

3. **Runtime layer** (`pkg/runtimes/`) - Container runtime abstraction via the `Runtime` interface in `runtime.go`. Only Docker is implemented (`pkg/runtimes/docker/`). The Docker runtime translates k3d types to Docker API calls (`translate.go`).

### Key Packages

- **`pkg/types/`** - Core domain types: `Cluster`, `Node`, `Registry`, `Role` (ServerRole, AgentRole, LoadBalancerRole, RegistryRole). Extensive label system (`k3d.cluster`, `k3d.role`, etc.) used to track Docker container metadata.
- **`pkg/config/`** - Versioned config system (current: `v1alpha5`, API version `k3d.io/v1alpha5`). Includes migration (`migrate.go`), JSON schema validation (`schema.json` embedded via `//go:embed`), and config processing/transformation pipeline.
- **`pkg/actions/`** - Node lifecycle hooks (e.g., `WriteFileAction`).
- **`pkg/logger/`** - Logging via logrus.

### Helper Components

- **`proxy/`** - nginx-based load balancer with confd for dynamic upstream config. Built as `ghcr.io/k3d-io/k3d-proxy`.
- **`tools/`** - Separate Go module (`tools/`) for the k3d-tools helper binary. Built as `ghcr.io/k3d-io/k3d-tools`.

## Config System

Config files are YAML with `apiVersion: k3d.io/v1alpha5` and `kind: Simple`. The processing pipeline: read YAML -> validate JSON schema -> migrate from older versions if needed -> transform to internal types -> merge with defaults. Config versions: v1alpha2 (legacy) -> v1alpha3 -> v1alpha4 -> v1alpha5 (current).

## Version & LDFLAGS

Version info is injected at build time via LDFLAGS into `version/version.go`:
- `version.Version` - git tag
- `version.K3sVersion` - latest stable k3s version (fetched from k3s update channel)
- `version.HelperVersionOverride` - optional override for helper image versions

## License & Contributing

This project is **MIT licensed** (Copyright 2019-2023 Thorsten Klein). All new Go source files must include the MIT copyright header present in existing files. See `CONTRIBUTING.md` for guidelines — check existing issues/PRs before opening new ones to avoid duplicates. The project follows the Contributor Covenant Code of Conduct (`CODE_OF_CONDUCT.md`).

## Conventions

- All Go source files have the MIT license copyright header
- Import alias: `k3d "github.com/k3d-io/k3d/v5/pkg/types"` is used throughout as the canonical import for the types package
- Docker is the only runtime, but all container operations go through the `Runtime` interface
- Node types are identified by `Role` and tracked via Docker container labels

## Issue & PR Handling

**Before starting work on any issue:**

1. **Check for duplicates** — search open and closed issues for the same topic (`gh issue list -S "keyword"` / `gh issue list -S "keyword" --state closed`)
2. **Check for existing PRs** — look for open PRs that already address the issue (`gh pr list -S "keyword"`)
3. **Check if it's a k3d issue** — many reports are actually k3s, Docker, or CNI issues. If so, redirect the reporter to the appropriate upstream repo
4. **Read the full thread** — issues often evolve through discussion; the original request may have been refined or scoped down

**When replying to issues:**

- For questions: answer concisely, point to env vars/flags/docs, close if resolved
- For bugs: confirm reproducibility scope, check if already fixed on `main`
- For features: check if there's a workaround, note if a PR would be welcome

**When working on PRs:**

1. **Check the linked issue** — understand the full context and any decisions made in discussion
2. **Check the PR's CI status** — don't start reviewing or building on a PR that's failing CI for unrelated reasons
3. **Check for related/conflicting PRs** — look for other open PRs touching the same files (`gh pr list` + review changed files)
4. **Verify the branch is up to date** with `main` before investing effort

**Scope awareness:**

- k3d only controls the container orchestration layer around k3s. Issues about k3s behavior, CNI plugins, Kubernetes internals, or Docker engine bugs are upstream concerns
- The proxy (`k3d-proxy`) and tools (`k3d-tools`) images are part of this project — issues about those are in scope
- Registry issues may involve the upstream `registry` image vs k3d's registry wiring — distinguish which layer is at fault

## Deep Dive References

For detailed analysis beyond this summary, see `.planning/codebase/`:

| Document | Contents |
|----------|----------|
| `STACK.md` | Full dependency inventory, Go version, build toolchain, Docker SDK usage |
| `ARCHITECTURE.md` | Layer diagrams, data flow through CLI->Client->Runtime, abstraction boundaries |
| `STRUCTURE.md` | Complete directory layout, file naming conventions, package organization |
| `CONVENTIONS.md` | Error handling patterns, logging conventions, code style, naming rules |
| `TESTING.md` | Unit test patterns, E2E test framework details, test helpers, fixture locations |
| `INTEGRATIONS.md` | Docker API interaction, k3s channel server, registry (wharfie), kubeconfig handling |
| `CONCERNS.md` | Technical debt, single-runtime limitation, config migration complexity, known fragile areas |
