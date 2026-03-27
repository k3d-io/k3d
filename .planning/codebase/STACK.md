# Technology Stack

**Analysis Date:** 2026-02-05

## Languages

**Primary:**
- Go 1.24.4 - Main CLI tool and core runtime (k3d executable)

**Secondary:**
- Shell (Bash) - Build scripts, test infrastructure, Docker entrypoints

## Runtime

**Environment:**
- Go 1.24.4 - Uses Go Modules for dependency management (`go.mod`)

**Package Manager:**
- Go Modules - Vendored dependencies via `/vendor` directory
- Lockfile: Present (`go.sum`)

## Frameworks

**Core:**
- Cobra v1.9.1 - CLI command framework (`github.com/spf13/cobra`)
- Viper v1.18.2 - Configuration management (`github.com/spf13/viper`)

**Testing:**
- Go `testing` package (standard library) - Unit tests
- Testify v1.10.0 - Assertions and mocking (`github.com/stretchr/testify`)
- httptest - HTTP mocking (standard library)

**Build/Dev:**
- Make - Primary build system (`Makefile` with `GOFLAGS=-mod=vendor`)
- Gox - Cross-platform compilation (`github.com/iwilltry42/gox` v0.1.0)
- golangci-lint v2.4.0 - Linting and code quality
- Docker BuildKit - Container image builds

## Key Dependencies

**Critical:**
- `github.com/docker/docker` v28.3.1 - Docker API client for container runtime operations
- `github.com/docker/go-connections` v0.5.0 - Docker network connection utilities
- `github.com/google/go-containerregistry` v0.20.6 - OCI/Docker image registry operations
- `github.com/rancher/wharfie` v0.6.2 - Registry configuration and endpoint management
- `k8s.io/client-go` v0.30.2 - Kubernetes client library (for kubeconfig operations)

**Logging & Utilities:**
- `github.com/sirupsen/logrus` v1.9.3 - Structured logging framework
- `golang.org/x/mod` v0.25.0 - Go module utilities
- `gopkg.in/yaml.v3` v3.0.1 - YAML parsing and serialization
- `k8s.io/utils` v0.0.0-20250604170112-4c0f3b243397 - Kubernetes utility functions

**Network & Registry:**
- `github.com/goodhosts/hostsfile` v0.1.6 - Host file manipulation
- `go4.org/netipx` v0.0.0-20231129151722-fdeea329fbba - Network IP utilities

**Infrastructure:**
- `github.com/mitchellh/go-homedir` v1.1.0 - Home directory detection
- `github.com/mitchellh/copystructure` v1.2.0 - Struct deep copying
- `github.com/imdario/mergo` v0.3.14 - Struct merging
- `github.com/liggitt/tabwriter` v0.0.0-20181228230101-89fcab3d43de - Table formatting

## Configuration

**Environment:**
- Uses Go Modules with vendored dependencies
- Build-time configuration via LDFLAGS:
  - `-X github.com/k3d-io/k3d/v5/version.Version=${GIT_TAG}` - Version from git tag
  - `-X github.com/k3d-io/k3d/v5/version.K3sVersion=${K3S_TAG}` - K3s version (fetched at build time from update.k3s.io)
  - `-X github.com/k3d-io/k3d/v5/version.HelperVersionOverride=${K3D_HELPER_VERSION}` - Optional helper image version override

**Build:**
- `Makefile` - Central build configuration
- `.golangci.yml` - Linting rules (typecheck disabled, common false positives configured)
- `docker-bake.hcl` - Multi-platform Docker image builds

**Enabled Build Flags:**
- `CGO_ENABLED=0` - Static binary compilation
- `GO111MODULE=on` - Go modules mode (explicit)
- `GOFLAGS=-mod=vendor` - Vendor dependencies enforcement

## Platform Requirements

**Development:**
- Go 1.24.4 or compatible version
- Docker/Docker daemon (for container operations)
- Make
- kubectl (for E2E tests)
- gox (for cross-compilation, installed via `make install-tools`)
- golangci-lint v2.4.0 (for linting, installed via `make install-tools`)

**Production:**
- Docker daemon (required for k3d to function - creates k3s clusters in containers)
- Linux kernel 3.10+ (Docker requirement)
- Architectures supported: `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/386`, `linux/arm`, `linux/arm64`, `windows/amd64`

**Docker Images Built:**
- Base image: `golang:1.24.4` (builder stage)
- Runtime images:
  - `docker:27.3.1-dind` - For E2E testing container
  - `nginx:1.27.0-alpine3.19` - For proxy service
  - `alpine:3.20` - For tools service

**K3s Integration:**
- k3s version pulled dynamically at build time from `https://update.k3s.io/v1-release/channels/stable`
- Default k3s version hardcoded: v1.32.5-k3s1
- k3s image tags use hyphen instead of plus (e.g., `v1.32.5-k3s1` instead of `v1.32.5+k3s1`)

---

*Stack analysis: 2026-02-05*
