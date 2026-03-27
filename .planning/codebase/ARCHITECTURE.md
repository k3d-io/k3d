# Architecture

**Analysis Date:** 2026-02-05

## Pattern Overview

**Overall:** Layered CLI application with adapter pattern for runtime abstraction

**Key Characteristics:**
- CLI commands orchestrate higher-level operations via Cobra framework
- Config system supports versioned schema with migrations (v1alpha2 through v1alpha5)
- Runtime abstraction decouples Docker implementation from core business logic
- Two-phase cluster lifecycle: preparation and creation/configuration via actions
- Plugin system extends functionality through subprocess execution

## Layers

**Command Layer (Presentation):**
- Purpose: Parse CLI arguments and invoke business logic via flags/options
- Location: `cmd/` directory with subpackages for each command domain
- Contains: Cobra command definitions, flag parsing, output formatting
- Depends on: Client layer for orchestration, config for parsing
- Used by: main.go entry point, plugin execution

**Config Management:**
- Purpose: Parse, validate, transform, and migrate user configuration files
- Location: `pkg/config/` with versioned subdirectories (v1alpha2-v1alpha5)
- Contains: Schema definitions, migration functions, transformation logic
- Depends on: Types (k3d domain models)
- Used by: Client layer for configuration input, config CLI commands

**Client Layer (Business Logic Orchestration):**
- Purpose: Orchestrate cluster/node operations using runtime and configuration
- Location: `pkg/client/` with ~4700 lines across 16 files
- Contains: ClusterRun, ClusterCreate, ClusterDelete, NodeAddToCluster, etc.
- Pattern: Public functions accept context, runtime, and domain objects
- Depends on: Runtime interface, Actions system, Types
- Used by: Command layer to execute business operations

**Runtime Abstraction:**
- Purpose: Provide platform-agnostic interface for container operations
- Location: `pkg/runtimes/` with interface in runtime.go and Docker implementation in docker/
- Contains: Runtime interface defining 30+ methods for container, network, volume operations
- Implementation: Docker implementation handles docker daemon interaction
- Pattern: SelectedRuntime variable allows runtime swapping; currently Docker-only
- Depends on: Docker SDK (docker/docker, docker/go-connections)

**Domain Types:**
- Purpose: Define k3d domain model entities (Cluster, Node, Network, Registry, etc.)
- Location: `pkg/types/` with specialized types in k3s/, k8s/, fixes/ subdirectories
- Contains: Core domain objects, labels/roles/statuses, and helper types
- Key abstractions: Cluster, Node, ClusterNetwork, Registry, NodeRole, LoadBalancer
- Depends on: No internal dependencies; foundational layer

**Actions System:**
- Purpose: Deferred execution on nodes after cluster infrastructure created
- Location: `pkg/actions/` with nodehooks.go implementing action types
- Contains: Action interface, WriteFileAction, RewriteFileAction implementations
- Pattern: Actions are collected during planning, executed during cluster start
- Depends on: Runtime, Types
- Used by: Client layer during cluster configuration

**Utilities:**
- Purpose: Shared helper functions across layers
- Location: `pkg/util/` for business logic utils; `cmd/util/` for CLI-specific utils
- Contains: YAML/filter operations, label/port parsing, registry helpers, file operations
- Used by: All layers as needed

## Data Flow

**Cluster Creation Flow:**

1. User invokes: `k3d cluster create -c config.yaml`
2. Command layer (`cmd/cluster/clusterCreate.go`) parses flags
3. Config system (`pkg/config/`) loads, validates, and transforms config file
4. Client orchestrator (`pkg/client/ClusterRun`) executes multi-phase process:
   - **Phase 0 - Preparation:** ClusterPrep creates network, image volume, tools node
   - **Phase 1 - Container Creation:** ClusterCreate invokes runtime to create node containers
   - **Phase 2 - Configuration:** Actions collected during prep, executed during node startup
   - **Phase 3 - Kubeconfig:** Cluster nodes used to generate kubeconfig
5. Runtime layer (`pkg/runtimes/docker/`) translates operations to Docker API calls
6. Registry/Load Balancer setup occurs via specialized client functions
7. Cluster returned to command layer for success output

**State Management:**
- Config state: YAML files on disk, versioned with migrations
- Runtime state: Docker containers, networks, volumes as source of truth
- Cluster metadata: Stored as Docker container labels (k3d.cluster, k3d.role, etc.)
- Actions state: Collected in-memory during planning, transient across lifecycle

## Key Abstractions

**Cluster:**
- Purpose: Represents a complete k3d Kubernetes cluster instance
- Examples: `pkg/types/types.go` defines Cluster struct with nodes, network, registry
- Pattern: Central aggregate root; all k3d operations centered on cluster identity
- Lifecycle: Create → Start/Stop → Delete

**Node:**
- Purpose: Represents a container running k3s or supportive software
- Examples: Server, Agent, LoadBalancer, Registry roles defined in types.go
- Pattern: Nodes belong to clusters or exist as shared resources (registry)
- Operations: Created via runtime, configured via actions, referenced by labels

**ClusterNetwork:**
- Purpose: Docker network shared by all nodes in a cluster
- Examples: Created with IP range, external flag for host access
- Pattern: One network per cluster; nodes connected on creation or addition

**Runtime:**
- Purpose: Abstract interface to container execution platform
- Examples: `pkg/runtimes/runtime.go` interface; `pkg/runtimes/docker/docker.go` implementation
- Pattern: Pluggable implementation; currently Docker-only
- Key methods: CreateNode, DeleteNode, StartNode, StopNode, ExecInNode, etc.

**Config:**
- Purpose: User-provided cluster specification
- Examples: SimpleConfig, ClusterConfig in v1alpha5 types
- Pattern: Versioned schema with migration chain; transforms to ClusterConfig for execution
- Features: Node filters for per-node customization, k3s args, volumes, ports, env vars

## Entry Points

**main.go:**
- Location: `/main.go`
- Triggers: `go run . cluster create -c config.yaml`
- Responsibilities: Package main; delegates to cmd.Execute()

**cmd.Execute():**
- Location: `cmd/root.go`
- Triggers: Called from main.go
- Responsibilities: Initializes Cobra root command, logging, runtime; handles plugin dispatch

**NewCmdK3d():**
- Location: `cmd/root.go`
- Triggers: Called during initialization
- Responsibilities: Creates root cobra.Command with all subcommands (cluster, node, image, registry, config, etc.)

**Cluster Commands:**
- Location: `cmd/cluster/clusterCreate.go` and siblings
- Triggers: `k3d cluster create`, `start`, `stop`, `delete`, `list`, `edit`
- Responsibilities: Parse cluster-specific flags, invoke client.ClusterRun or equivalent

**Config Commands:**
- Location: `cmd/config/`
- Triggers: `k3d config init`, `view`, `migrate`
- Responsibilities: Configuration file management and schema migration

## Error Handling

**Strategy:** Error wrapping with context using fmt.Errorf, logged via l.Log()

**Patterns:**
- Errors propagate up from runtime through client to command layer
- Fatal errors logged and cause os.Exit(1) via l.Log().Fatalln()
- Non-fatal errors returned with context wrapped as `fmt.Errorf("message: %w", err)`
- Runtime errors wrapped with specific error types in `pkg/runtimes/errors/`
- Validation errors returned early from config transformation phase

## Cross-Cutting Concerns

**Logging:** Logrus instance via singleton `pkg/logger/logger.go`; hooks send info/debug/trace to stdout, errors/warnings to stderr

**Validation:** Config validation in `pkg/config/validate.go` against JSON schema; argument validation in command definitions

**Authentication:** Docker runtime uses DOCKER_HOST env var or default socket path; no explicit k3s auth (managed by cluster runtime)

**Networking:** Docker networks created per cluster; IPAM handled by Docker; host port mapping via nat.Port (docker/go-connections)

**Labels:** Docker container labels store cluster metadata for discovery (k3d.cluster, k3d.role, etc.); queried via GetNodesByLabel

---

*Architecture analysis: 2026-02-05*
