# External Integrations

**Analysis Date:** 2026-02-05

## APIs & External Services

**Docker Daemon:**
- What it's used for: Container runtime for creating and managing k3s cluster nodes and registries
  - SDK/Client: `github.com/docker/docker` v28.3.1
  - Connection: Unix socket (default: `/var/run/docker.sock`) or `DOCKER_HOST` environment variable
  - Features: Container lifecycle, network creation, volume management, image operations

**Container Registry:**
- What it's used for: OCI image pulling, pushing, and manipulation
  - SDK/Client: `github.com/google/go-containerregistry` v0.20.6
  - Usage: Image metadata inspection, layer operations via Crane CLI bindings

**Registry Configuration Service:**
- What it's used for: Registry endpoint resolution and proxy configuration
  - SDK/Client: `github.com/rancher/wharfie` v0.6.2
  - Purpose: Parse and manage container registry endpoints and mirrors

## Data Storage

**Databases:**
- None (k3d is a CLI tool, not a service with persistent storage)

**File Storage:**
- Local filesystem only
- Key paths:
  - Docker socket: `/var/run/docker.sock`
  - Default kubeconfig: `$HOME/.kube/config` or path from `$KUBECONFIG` env var
  - Registry YAML config: `/etc/rancher/k3s/registries.yaml` (inside k3s containers)
  - Registry data mount: `/var/lib/registry` (inside registry containers)
  - Temporary kubeconfig ConfigMap path: `/tmp/localRegistryHostingCM.yaml`

**Caching:**
- None - k3d operates as a CLI tool without caching layer

## Authentication & Identity

**Auth Provider:**
- Custom implementation
  - Kubeconfig management via `k8s.io/client-go` v0.30.2
  - Loads kubeconfig from standard locations or `KUBECONFIG` environment variable
  - Creates kubeconfig entries for created clusters

**Docker Authentication:**
- Uses host Docker daemon authentication
- Supports Docker credential helpers via docker/docker CLI integration
- Registry authentication handled by google/go-containerregistry (supports Docker credential chain)

## Monitoring & Observability

**Error Tracking:**
- None configured

**Logs:**
- Structured logging via `github.com/sirupsen/logrus` v1.9.3
- Log level controlled via CLI flags
- E2E tests use custom log level via `LOG_LEVEL` environment variable
- Node wait logs configurable via `K3D_LOG_NODE_WAIT_LOGS` environment variable

## CI/CD & Deployment

**Hosting:**
- GitHub (public repository)
- Container images published to: `ghcr.io/k3d-io/` (GitHub Container Registry)

**CI Pipeline:**
- GitHub Actions (workflows in `.github/workflows/`)
- Key workflows:
  - `release.yaml` - Test, build, and release on push/PR
  - `test-matrix.yaml` - Multi-platform testing
  - `test-install-osmatrix.yaml` - Installation testing
  - `aur-release.yaml` - Arch Linux AUR release

**Build & Publish:**
- Docker Buildx for multi-platform builds (linux/amd64, linux/arm64, linux/arm/v7)
- Docker image metadata via GitHub Actions Docker/metadata-action
- Images published on tags

## Environment Configuration

**Required env vars (Runtime):**
- `DOCKER_HOST` - Override Docker daemon socket/connection
- `DOCKER_SOCK` - Override Docker socket path (alternative to DOCKER_HOST)
- `KUBECONFIG` - Kubernetes configuration file path(s)
- `K3S_URL` - (Inside cluster nodes) K3s API endpoint
- `K3S_TOKEN` - (Inside cluster nodes) K3s cluster token
- `K3S_KUBECONFIG_OUTPUT` - (Inside cluster nodes) Output path for kubeconfig

**Optional env vars (Configuration):**
- `K3D_IMAGE_LOADBALANCER` - Override load balancer image
- `K3D_IMAGE_TOOLS` - Override tools image
- `K3D_HELPER_IMAGE_TAG` - Override helper image tag
- `K3D_LOG_NODE_WAIT_LOGS` - Enable detailed node wait logs for specific roles
- `K3D_DEBUG_COREDNS_RETRIES` - CoreDNS retry configuration
- `K3D_DEBUG_DISABLE_DOCKER_INIT` - Disable Docker initialization
- `K3D_DEBUG_NODE_WAIT_BACKOFF_LIMIT` - Node wait backoff tuning
- `K3D_FIX_CGROUPV2` - Enable cgroup v2 fix
- `K3D_FIX_DNS` - Enable DNS fix
- `K3D_FIX_MOUNTS` - Enable mount fix
- `XDG_CONFIG_HOME` - Base config directory (Linux)
- `WSL_DISTRO_NAME` - Used to detect WSL environment

**Build-time env vars:**
- `GIT_TAG_OVERRIDE` - Override git tag for versioning
- `K3D_HELPER_VERSION` - Set helper image version
- `E2E_*` - E2E test configuration (various options)

**Secrets location:**
- Docker credential helpers (standard Docker auth)
- No dedicated secrets storage - relies on Docker daemon and Kubernetes RBAC

## Webhooks & Callbacks

**Incoming:**
- None

**Outgoing:**
- None

## External Service Dependencies

**K3s Release Channel API:**
- URL: `https://update.k3s.io/v1-release/channels/stable`
- Purpose: Fetch latest stable k3s version at build time
- Used in: `Makefile` via curl to determine `K3S_TAG`
- Fallback: Hardcoded version `v1.32.5-k3s1` in `version/version.go`

**K3s Version Resolution API:**
- Purpose: Resolve specific k3s version channels to exact versions at runtime
- Package: `version.GetK3sVersion()` in `version/version.go`
- Used by: CLI version command and cluster initialization

## Image Sources

**Container Images Used:**
- `k3s:[version]` - Main K3s image (pulled from configured registries)
- `rancher/coredns:[version]` - CoreDNS (K3s component)
- `rancher/klipper-lb:[version]` - ServiceLB/load balancer (K3s component)
- `rancher/local-path-provisioner:[version]` - Storage provisioner (K3s component)
- Custom k3d images:
  - `k3d-proxy:[tag]` - Built from `ghcr.io/k3d-io/k3d-proxy`
  - `k3d-tools:[tag]` - Built from `ghcr.io/k3d-io/k3d-tools`

## Registry Configuration

**Default Registry:**
- Docker Hub: `registry-1.docker.io`
- Alternative registries: Configurable via registries.yaml config files
- Local registry support: Built-in k3d registry node with configurable proxy options

**Registry Proxy Options:**
- Remote URL rewriting
- Username/password authentication for proxying
- Delete operation enablement flag
- Port matching enforcement between internal and external ports

---

*Integration audit: 2026-02-05*
