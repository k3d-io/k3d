# Codebase Structure

**Analysis Date:** 2026-02-05

## Directory Layout

```
k3d/
├── cmd/                    # CLI commands layer (Cobra-based)
│   ├── cluster/           # cluster create/start/stop/delete/list/edit commands
│   ├── node/              # node create/start/stop/delete/list/edit commands
│   ├── registry/          # registry create/start/stop/delete/list commands
│   ├── image/             # image import commands
│   ├── kubeconfig/        # kubeconfig get/merge commands
│   ├── config/            # config init/view/migrate commands
│   ├── debug/             # debug utilities
│   ├── util/              # CLI-specific utilities (filters, ports, listings)
│   └── root.go            # Root command, logging init, runtime init
│
├── pkg/                   # Core business logic (reusable library)
│   ├── client/            # Orchestration layer (4700+ lines)
│   │   ├── cluster.go     # ClusterRun, ClusterCreate, ClusterDelete, etc.
│   │   ├── node.go        # NodeAddToCluster, node operations
│   │   ├── registry.go    # Registry creation/management
│   │   ├── loadbalancer.go
│   │   ├── kubeconfig.go  # Kubeconfig generation/merging
│   │   ├── ports.go       # Port mapping operations
│   │   ├── hooks.go       # Cluster lifecycle hooks
│   │   ├── tools.go       # Tools node management
│   │   ├── environment.go # Cluster environment info gathering
│   │   ├── host.go        # Host interaction helpers
│   │   ├── ipam.go        # IP address management
│   │   └── *_test.go      # Unit tests
│   │
│   ├── config/            # Configuration management
│   │   ├── config.go      # Config loader (FromViper)
│   │   ├── v1alpha2/      # Legacy schema and migrations
│   │   ├── v1alpha3/
│   │   ├── v1alpha4/
│   │   ├── v1alpha5/      # Current default schema
│   │   │   ├── types.go   # SimpleConfig, ClusterConfig structs
│   │   │   └── schema.json
│   │   ├── types/         # Config interface definitions
│   │   ├── migrate.go     # Cross-version migration logic
│   │   ├── transform.go   # Config → internal types transformation
│   │   ├── validate.go    # Schema validation
│   │   ├── merge.go       # Config merging for multiple files
│   │   ├── process.go     # Config processing pipeline
│   │   └── jsonschema.go
│   │
│   ├── types/             # Domain model types
│   │   ├── types.go       # Cluster, Node, ClusterNetwork, Registry, LoadBalancer
│   │   ├── node.go        # Node role and status enums
│   │   ├── loadbalancer.go
│   │   ├── registry.go
│   │   ├── images.go
│   │   ├── intent.go
│   │   ├── env.go         # Environment variable defaults
│   │   ├── files.go       # File upload structures
│   │   ├── defaults.go    # Default values
│   │   ├── k3s/           # k3s-specific types (args, env, channel, paths)
│   │   ├── k8s/           # Kubernetes-specific types
│   │   └── fixes/         # Bug fixes and workarounds
│   │
│   ├── runtimes/          # Container runtime abstraction
│   │   ├── runtime.go     # Runtime interface (30+ methods)
│   │   ├── docker/        # Docker implementation
│   │   │   ├── docker.go  # Docker struct, ID(), GetHost()
│   │   │   ├── container.go
│   │   │   ├── network.go
│   │   │   ├── volume.go
│   │   │   ├── image.go
│   │   │   ├── node.go
│   │   │   ├── kubeconfig.go
│   │   │   ├── translate.go
│   │   │   ├── host.go
│   │   │   ├── util.go
│   │   │   ├── machine.go # Docker-machine support
│   │   │   ├── types.go
│   │   │   ├── info.go
│   │   │   └── *_test.go
│   │   ├── types/         # Runtime type definitions
│   │   ├── errors/        # Runtime-specific error types
│   │   └── util/          # Shared runtime utilities (volumes)
│   │
│   ├── actions/           # Deferred node configuration actions
│   │   └── nodehooks.go   # WriteFileAction, RewriteFileAction
│   │
│   ├── logger/            # Logging abstraction
│   │   └── logger.go      # Logrus singleton
│   │
│   └── util/              # Shared utilities
│       ├── yaml.go        # YAML encoding/decoding
│       ├── filter.go      # Node filtering
│       ├── labels.go      # Label parsing
│       ├── ports.go       # Port parsing and validation
│       ├── registry.go    # Registry helpers
│       ├── files.go       # File operations
│       └── *_test.go
│
├── main.go                # Single-line entry point
├── go.mod                 # Go module definition (v1.24.4)
├── go.sum                 # Go dependency checksums
├── Makefile               # Build targets
├── Dockerfile             # Container image for k3d
├── docker-bake.hcl        # Docker buildkit configuration
├── .golangci.yml          # Linter config
│
├── tests/                 # Integration tests
│   └── assets/
│
├── docs/                  # Documentation
│   ├── design/
│   ├── faq/
│   ├── usage/
│   └── static/
│
├── scripts/               # Build and utility scripts
├── tools/                 # Tool packages (docgen, version)
├── version/               # Version information
├── docgen/                # Documentation generation
└── proxy/                 # k3d-proxy service (load balancer)
    ├── conf.d/
    ├── templates/
    └── test/
```

## Directory Purposes

**cmd/:**
- Purpose: CLI command definitions and handlers
- Contains: Cobra command structs, flag parsing, output formatting
- Pattern: One subpackage per command group (cluster, node, image, etc.)
- Key files: `root.go` orchestrates command initialization and runtime setup

**pkg/client/:**
- Purpose: Business logic orchestration for cluster/node operations
- Contains: 16 files, ~4700 lines, public functions for major operations
- Pattern: Client functions accept context, runtime, config/domain objects
- Examples: ClusterRun orchestrates full creation; ClusterCreate handles container phase

**pkg/config/:**
- Purpose: Parse, validate, transform user configuration
- Contains: Versioned schema packages (v1alpha2-v1alpha5) with migrations
- Pattern: FromViper loads from viper.Viper; each version has GetConfigByKind
- Key logic: v1alpha5 is current; migrations handle version upgrades

**pkg/types/:**
- Purpose: Domain model definitions for k3d concepts
- Contains: Cluster, Node, Registry, LoadBalancer, ClusterNetwork structs
- Pattern: Foundational layer; no internal dependencies; uses only standard lib
- Labels: 20+ constants (k3d.cluster, k3d.role, etc.) stored as Docker labels

**pkg/runtimes/:**
- Purpose: Abstract container runtime operations
- Contains: Interface (runtime.go) and Docker implementation (docker/ subpackage)
- Pattern: SelectedRuntime variable allows switching; interface-based design
- Extensibility: Can add containerd, cri-o implementations by conforming to Runtime interface

**pkg/actions/:**
- Purpose: Deferred actions executed on cluster nodes
- Contains: Action interface, WriteFileAction, RewriteFileAction
- Pattern: Actions collected during planning, executed during cluster start
- Usage: Used for kubeconfig setup, registries config, etc.

**pkg/logger/:**
- Purpose: Logging abstraction
- Contains: Logrus singleton initialized in cmd/root.go
- Pattern: Centralized through l.Log() convenience function

**pkg/util/:**
- Purpose: Shared utilities across layers
- Contains: YAML helpers, label/port parsing, filter logic, file operations
- Pattern: Pure utility functions; no state or dependencies on business logic

**cmd/util/:**
- Purpose: CLI-specific utilities
- Contains: Filter/listing output, port handling for CLI, config utilities
- Pattern: Higher-level CLI utilities; reuses pkg/util where possible

**proxy/:**
- Purpose: Load balancer and registry proxy service
- Contains: NGINX-based proxy configuration
- Status: Used by k3d to provide load balancing for k3s clusters

**tests/:**
- Purpose: Integration tests
- Contains: Test fixtures and assets
- Status: Limited test coverage for integration scenarios

**docs/:**
- Purpose: User documentation
- Contains: Design docs, FAQ, usage guides, static assets

## Key File Locations

**Entry Points:**
- `main.go`: Single entry point; imports github.com/k3d-io/k3d/v5/cmd and calls cmd.Execute()
- `cmd/root.go`: Creates root Cobra command, initializes logging and runtime
- `cmd/cluster/clusterCreate.go`: Entry point for cluster creation command

**Configuration:**
- `pkg/config/config.go`: Config loader (FromViper function)
- `pkg/config/v1alpha5/types.go`: Current schema types (SimpleConfig, ClusterConfig)
- `pkg/config/v1alpha5/schema.json`: JSON schema for validation (embedded)
- `pkg/config/migrate.go`: Cross-version migration logic

**Core Logic:**
- `pkg/client/cluster.go`: ClusterRun (main orchestrator), ClusterCreate, ClusterDelete, etc.
- `pkg/client/node.go`: NodeAddToCluster, node lifecycle operations
- `pkg/types/types.go`: Domain model (Cluster, Node, ClusterNetwork, Registry)

**Testing:**
- `pkg/client/*_test.go`: Unit tests for client functions
- `pkg/config/*_test.go`: Config parsing and migration tests
- `cmd/util/ports_test.go`: Port parsing tests
- `tests/`: Integration test assets

**Runtime:**
- `pkg/runtimes/runtime.go`: Runtime interface definition
- `pkg/runtimes/docker/docker.go`: Docker implementation entry point
- `pkg/runtimes/docker/container.go`: Container operation implementations

## Naming Conventions

**Files:**
- `<verb>.go`: Command implementations (e.g., `clusterCreate.go`, `clusterDelete.go`)
- `<noun>.go`: Data handling (e.g., `cluster.go`, `node.go`, `registry.go`)
- `*_test.go`: Unit tests (e.g., `config_test.go`)

**Directories:**
- `v1alpha*/`: Versioned API packages following semantic versioning
- `docker/`: Implementation-specific packages (currently only docker)
- `k3s/`, `k8s/`: Domain-specific type groupings

**Functions:**
- `Cluster<Operation>`: Cluster operations (ClusterCreate, ClusterDelete, ClusterStart)
- `Node<Operation>`: Node operations (NodeCreate, NodeDelete, NodeAddToCluster)
- `<Type><Operation>`: Registry/LB operations (RegistryCreate, LoadBalancerStart)
- `New<Type>`: Constructor functions (NewCmdCluster, NewCmdNode)

**Types:**
- `Cluster`, `Node`, `ClusterNetwork`: Domain entities (PascalCase)
- `<Type>CreateOpts`, `<Type>DeleteOpts`: Operation option structs
- `<Type>Config`: Configuration structs in config package

## Where to Add New Code

**New Cluster Operation (e.g., cluster update):**
- Command handler: `cmd/cluster/clusterUpdate.go`
- Add subcommand in: `cmd/cluster/cluster.go` (NewCmdCluster)
- Business logic: `pkg/client/cluster.go` (add ClusterUpdate function)
- Test coverage: `pkg/client/cluster_test.go`

**New Node Type (beyond server/agent/loadbalancer/registry):**
- Add Role constant to: `pkg/types/types.go` (NodeRole enum)
- Add creation logic to: `pkg/client/node.go` (NodeCreate)
- Add runtime support to: `pkg/runtimes/docker/node.go`

**New Configuration Option:**
- Add field to: `pkg/config/v1alpha5/types.go` (SimpleConfig or substructure)
- Update schema: `pkg/config/v1alpha5/schema.json` (update embedded JSON)
- Add migration if needed: `pkg/config/v1alpha5/migrations.go`
- Add processing: `pkg/config/process.go` (transform to internal representation)

**Utility Function:**
- Shared logic (non-CLI): `pkg/util/<domain>.go` (e.g., pkg/util/labels.go)
- CLI-specific: `cmd/util/<function>.go`
- Tests co-located: `pkg/util/<domain>_test.go`

**Runtime Implementation Change:**
- Docker-specific: `pkg/runtimes/docker/<component>.go`
- Interface change: `pkg/runtimes/runtime.go`
- Keep Docker as sole implementation; note extensibility point

## Special Directories

**vendor/:**
- Purpose: Vendored dependencies
- Generated: Yes (go mod vendor)
- Committed: Yes

**bin/:**
- Purpose: Build output directory
- Generated: Yes (make build)
- Committed: No

**docker-bake.hcl:**
- Purpose: Docker buildkit configuration for multi-platform builds
- Manual or auto-generated: Manual configuration

**go.work, .local/**
- Purpose: Go workspace for local development
- Status: For tools and utilities separate from main module

---

*Structure analysis: 2026-02-05*
