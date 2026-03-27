# Coding Conventions

**Analysis Date:** 2026-02-05

## Language & Version

**Primary Language:** Go 1.24.4

All code follows standard Go conventions and idioms. Copyright headers are applied to source files (MIT License).

## Naming Patterns

**Files:**
- Lowercase with underscores for multi-word names: `ports_test.go`, `translate_test.go`
- Test files: `{name}_test.go` (co-located with implementation)
- Test data: `./test_assets/` subdirectory (e.g., `test_assets/config_test_simple.yaml`)
- Private helpers: `prepCreateLocalRegistryHostingConfigMap` (lowercase first letter)

**Functions:**
- PascalCase for exported functions: `ClusterRun()`, `ClusterCreate()`, `NodeCreateMulti()`, `ParsePortExposureSpec()`
- camelCase for unexported functions: `populateClusterFieldsFromLabels()`, `prepCreateLocalRegistryHostingConfigMap()`
- Descriptive names indicating action and subject: `ClusterDelete()`, `ClusterList()`, `GetNodesByLabel()`
- Test functions: `Test{FunctionName}_{Scenario}()` format, e.g., `Test_ParsePortExposureSpec_PortMatchEnforcement()`

**Variables & Constants:**
- PascalCase for exported: `ServerRole`, `AgentRole`, `LoadBalancerRole`
- camelCase for unexported: `apiPortRegexp`, `clusterPrepCtx`, `expectedYAMLString`
- Constants: `UPPER_CASE` with underscores (used for validation constants, error messages, and labels): `LabelClusterName`, `LabelServerAPIPort`, `K3dEnvDebugDisableDockerInit`
- Config variables: camelCase for struct fields with JSON/YAML tags: `DisableImageVolume`, `WaitForServer`, `Timeout`

**Types & Interfaces:**
- PascalCase for exported structs and interfaces: `Cluster`, `Node`, `Registry`, `Runtime`
- Descriptive struct field names with struct tags for serialization: `DisableLoadbalancer`, `GlobalLabels`, `NodeHooks`
- Role types use string-based constants with type definition: `type Role string` with constants like `ServerRole = "server"`
- Options structs use `Opts` suffix: `ClusterCreateOpts`, `ClusterStartOpts`, `NodeCreateOpts`

## Code Organization

**Import Organization:**
Files follow standard Go import grouping:
1. `import (` - standard library imports (e.g., `context`, `fmt`, `os`)
2. Third-party imports (e.g., `github.com/`, external libraries)
3. Internal k3d imports (e.g., `github.com/k3d-io/k3d/v5/pkg/...`)

**Import Aliases:**
- Consistent use of short aliases for frequently used imports:
  - `l "github.com/k3d-io/k3d/v5/pkg/logger"` (logger)
  - `k3d "github.com/k3d-io/k3d/v5/pkg/types"` (types)
  - `k3drt "github.com/k3d-io/k3d/v5/pkg/runtimes"` (runtimes)
  - `config "github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"` (config)
  - `cliutil "github.com/k3d-io/k3d/v5/cmd/util"` (command utilities)

**Module Path Aliases:**
- All imports use full module path: `module github.com/k3d-io/k3d/v5`

## Error Handling

**Patterns:**
- Use `fmt.Errorf()` for error wrapping with context and formatting: `fmt.Errorf("Failed Cluster Creation: %+v", err)`
- Chain errors with `%w` verb for error unwrapping: `fmt.Errorf("failed to gather environment information: %w", err)`
- Include operation context in error messages: `fmt.Errorf("error creating temp hosts file: %w", err)`
- Return early on error within functions
- Use `error` as last return value for multi-return functions

**Examples from codebase:**
```go
if err := ClusterPrep(ctx, runtime, clusterConfig); err != nil {
    return fmt.Errorf("Failed Cluster Preparation: %+v", err)
}

if err != nil {
    return envInfo, fmt.Errorf("failed to get host IP: %w", err)
}
```

**Validation:**
- Validation functions return descriptive errors: `fmt.Errorf("Invalid cluster name. %+v", ValidateHostname(name))`
- Separate validation concern with error context

## Logging

**Framework:** `github.com/sirupsen/logrus`

**Access Pattern:**
- Accessed via `l.Log()` (alias to logger package)
- Call as method on returned logger instance: `l.Log().Infof(...)`, `l.Log().Warnf(...)`, `l.Log().Debugf(...)`

**Log Levels Used:**
- `Infof()`: General informational messages (e.g., "Using the k3d-tools node to gather environment information")
- `Warnf()`: Warning conditions (e.g., "Failed to delete tools node '%s'. This is not critical...")
- `Debugf()`: Detailed debugging info (e.g., "Adding node %s to cluster %s based on existing node %s")
- `Tracef()`: Most detailed trace logging (e.g., "inspecting port mapping for %s with nodefilters %s")
- `Errorf()`: Error conditions (e.g., "error printing loadbalancer config: %v")

**Patterns:**
```go
l.Log().Infof("Using the k3d-tools node to gather environment information")
l.Log().Debugf("Didn't find node with role '%s' in cluster '%s'. Choosing any other node...", node.Role, cluster.Name)
l.Log().Tracef("Sanitized Source Node: %+v\nNew Node: %+v", srcNode, node)
```

**Conditional Logging:**
- Check log level before expensive operations: `if l.Log().GetLevel() >= logrus.DebugLevel { ... }`

## Comments

**Documentation Comments:**
- Public functions and types should have descriptive comment preceding declaration
- Comments use complete sentences
- All type constants are commented: `// ServerRole ...`

**Inline Comments:**
- Used sparingly for non-obvious logic
- Explain "why", not "what" (code is self-documenting for "what")

**TODOs and FIXMEs:**
- Used to mark known issues and future work
- Include context about what needs to be done (found throughout codebase)
- Examples: `// TODO: move "proxy" and "direct" allowed suffices to constants`
- Examples: `// FIXME: arbitrary wait for one second to avoid race conditions`

## Function Design

**Size Guidelines:**
- Functions typically 50-150 lines including comments
- Larger functions break down logical steps with comments
- Example: `ClusterRun()` in `pkg/client/cluster.go` clearly sections steps with comments

**Parameters:**
- Use context.Context as first parameter for functions involving I/O or runtime operations
- Typed structs preferred over multiple primitive parameters (e.g., `*k3d.ClusterCreateOpts` instead of multiple booleans)
- Pass values by pointer for mutable types: `*k3d.Cluster`, `*k3d.Node`
- Pass by value for immutable small types: `k3d.Role`

**Return Values:**
- Functions performing I/O return `error` as last return value
- Query functions return `(T, error)` where T is the result
- Multiple return values used for status/metadata: `(running bool, status string, error)`

**Example Patterns:**
```go
func ClusterRun(ctx context.Context, runtime k3drt.Runtime, clusterConfig *config.ClusterConfig) error
func ClusterList(ctx context.Context, runtime k3drt.Runtime) ([]*k3d.Cluster, error)
func GetNodeStatus(ctx context.Context, *k3d.Node) (bool, string, error) // returns (running, status, error)
```

## Struct Design

**Field Naming:**
- Exported fields use PascalCase: `Name`, `Role`, `Image`, `Volumes`
- Struct tags for serialization (JSON, YAML, mapstructure): `json:"name,omitempty"`, `mapstructure:"ip"`
- Omit empty fields in JSON/YAML: `,omitempty` tag convention

**Composition:**
- Embed types for shared behavior (e.g., `TypeMeta`, `ObjectMeta` in config types)
- Use nested structs for grouping related fields: `Registries struct { Create, Use, Config }`
- Options structs contain configuration: `ClusterCreateOpts`, `ClusterStartOpts`

**Examples:**
```go
type ClusterCreateOpts struct {
    DisableImageVolume  bool              `json:"disableImageVolume,omitempty"`
    WaitForServer       bool              `json:"waitForServer,omitempty"`
    Timeout             time.Duration     `json:"timeout,omitempty"`
    GlobalLabels        map[string]string `json:"globalLabels,omitempty"`
}

type HostAlias struct {
    IP        string   `mapstructure:"ip" json:"ip"`
    Hostnames []string `mapstructure:"hostnames" json:"hostnames"`
}
```

## Interface Design

**Pattern:**
- Interfaces define behavior contracts for pluggable implementations
- Example: `Runtime` interface in `pkg/runtimes/runtime.go` defines all container runtime operations
- Methods on interfaces follow action-first naming: `GetNodesByLabel()`, `CreateNetworkIfNotPresent()`, `DisconnectNodeFromNetwork()`
- Interface methods include context.Context as first parameter where applicable

## Package Structure

**Package Organization:**
- `cmd/`: CLI command implementations (cluster, node, config, registry, kubeconfig, debug, image subcommands)
- `cmd/util/`: CLI utilities (config handling, port parsing, plugins)
- `pkg/client/`: Core API client logic (cluster, node, registry operations)
- `pkg/config/`: Configuration parsing, validation, transformation
- `pkg/types/`: Type definitions and constants
- `pkg/runtimes/`: Runtime abstraction and implementations
- `pkg/util/`: Shared utilities (YAML parsing, etc.)
- `pkg/logger/`: Logging setup
- `pkg/actions/`: Action definitions for node lifecycle hooks
- `version/`: Version information

---

*Convention analysis: 2026-02-05*
