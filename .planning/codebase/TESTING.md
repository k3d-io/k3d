# Testing Patterns

**Analysis Date:** 2026-02-05

## Test Framework

**Runner:**
- Standard Go `testing` package
- Config: Built-in, no external test config file required
- Dependencies: `github.com/stretchr/testify` v1.10.0

**Assertion Libraries:**
- `github.com/stretchr/testify/require` - Primary assertion library for enforcing test conditions
- `gotest.tools/assert` - Secondary library for table-driven test assertions
- `github.com/go-test/deep` - Deep equality comparisons for complex structures

**Run Commands:**
```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./pkg/config

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Available via Makefile
make test              # Run unit tests with $(TESTFLAGS) variable
make e2e               # Run end-to-end tests with custom environment variables
make ci-tests          # Run fmt check, lint, and e2e tests in CI pipeline
```

## Test File Organization

**Location Pattern:**
- Co-located with implementation files in same package
- Test files use `_test.go` suffix: `ports_test.go`, `cluster.go` + `cluster_test.go`
- Located in same directory as source files

**Directory Structure:**
```
pkg/config/
├── config.go
├── config_test.go
├── process.go
├── process_test.go
├── validate_test.go
├── test_assets/
│   ├── config_test_simple.yaml
│   ├── config_test_cluster.yaml
│   ├── config_test_registries.yaml
│   ├── ns-baz.yaml
│   └── ... (other YAML fixtures)

pkg/util/
├── yaml.go
├── yaml_test.go

cmd/util/
├── ports.go
├── ports_test.go

pkg/client/
├── cluster.go
├── cluster_test.go
├── registry.go
├── registry_test.go
```

**Test Fixtures Location:**
- YAML/data fixtures stored in `test_assets/` subdirectory next to tests
- Examples: `pkg/config/test_assets/config_test_simple.yaml`
- Referenced by relative path: `./test_assets/config_test_simple.yaml`

## Test Structure

**Naming Convention:**
Functions follow pattern `Test{FunctionName}_{Scenario}()`:
- `TestReadSimpleConfig()` - basic functionality
- `Test_ParsePortExposureSpec_PortMatchEnforcement()` - specific scenario
- `TestValidateClusterConfig()` - validation tests
- `TestProcessClusterConfig()` - processing tests

**Basic Test Function Structure:**
```go
func TestValidateClusterConfig(t *testing.T) {
    cfgFile := "./test_assets/config_test_cluster.yaml"

    vip := viper.New()
    vip.SetConfigFile(cfgFile)
    _ = vip.ReadInConfig()

    cfg, err := FromViper(vip)
    if err != nil {
        t.Error(err)
    }

    if err := ValidateClusterConfig(context.Background(), runtimes.Docker, cfg.(conf.ClusterConfig)); err != nil {
        t.Error(err)
    }
}
```

**Test Assertion Patterns:**
- Use `require` for critical assertions that should fail test:
  ```go
  require.Nil(t, err)
  require.Equal(t, string(r.Port), "1111/tcp")
  require.NotEqual(t, strings.Split(string(r.Port), "/")[0], string(r.Binding.HostPort))
  require.NoError(t, err)
  ```

- Use `assert` for informational assertions:
  ```go
  assert.Assert(t, clusterCfg.ClusterCreateOpts.DisableLoadBalancer == false, "The load balancer should be enabled")
  assert.Equal(t, testSet.expected[idx], actual[idx])
  assert.NilError(t, err)
  ```

## Table-Driven Tests

**Pattern:**
Tests use map-based table-driven pattern for multiple scenarios:

```go
func TestSplitYAML(t *testing.T) {
    testSets := map[string]struct {
        document string
        expected []string
    }{
        "single": {
            document: `name: clusterA`,
            expected: []string{`name: clusterA`},
        },
        "multiple": {
            document: `name: clusterA
---
name: clusterB
`,
            expected: []string{`name: clusterA`, `name: clusterB`},
        },
    }

    for name, testSet := range testSets {
        t.Run(name, func(t *testing.T) {
            actual, err := SplitYAML([]byte(testSet.document))
            assert.NilError(t, err)
            assert.Equal(t, len(testSet.expected), len(actual))
            for idx := range testSet.expected {
                assert.Equal(t, testSet.expected[idx], strings.TrimSpace(string(actual[idx])))
            }
        })
    }
}
```

**Benefits:**
- Multiple test cases in single function with descriptive names
- Reduces code duplication
- Easy to add new test cases

## Context Usage

**Pattern:**
Tests requiring I/O pass `context.Background()` to functions:
```go
clusterCfg, err := TransformSimpleToClusterConfig(context.Background(), runtimes.Docker, cfg.(conf.SimpleConfig), cfgFile)
```

## Test Data & Fixtures

**YAML Fixtures:**
- Located in `test_assets/` subdirectories
- Named with pattern `{feature}_test_{scenario}.yaml`
- Examples:
  - `config_test_simple.yaml` - simple cluster config
  - `config_test_cluster.yaml` - full cluster config
  - `config_test_registries.yaml` - registry configuration
  - Migration test files: `config_test_simple_migration_v1alpha{2,3,4,5}.yaml`

**Inline Test Data:**
- Small test data created inline in test functions
- Struct construction for complex types:
  ```go
  inputNode := &k3d.Node{
      Name:    "test",
      Role:    k3d.ServerRole,
      Image:   "rancher/k3s:v0.9.0",
      Volumes: []string{"/test:/tmp/test"},
      Env:     []string{"TEST_KEY_1=TEST_VAL_1"},
      Ports: nat.PortMap{
          "6443/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "6443"}},
      },
  }
  ```

**Test Output Logging:**
- Use `t.Logf()` for detailed test execution info:
  ```go
  t.Logf("\n========== Read Config and transform to cluster ==========\n%+v\n=================================\n", cfg)
  t.Logf("\n===== Resulting Cluster Config =====\n%+v\n===============\n", clusterCfg)
  ```

## Error Testing

**Pattern:**
Tests check both success and error conditions:
```go
// Success case
r, err := ParsePortExposureSpec("9999", "1111", false)
require.Nil(t, err)
require.Equal(t, string(r.Port), "1111/tcp")

// Alternative error case
r, err = ParsePortExposureSpec("random", "1", false)
require.Nil(t, err)  // No error expected
require.NotEqual(t, ...)  // But result should differ
```

**Error Assertion:**
```go
if err != nil {
    t.Error(err)  // Fail test with error message
}

if err := SomeFunc(); err != nil {
    t.Error(err)
}

require.NoError(t, err)  // Assert no error occurred
assert.NilError(t, err)  // Assert no error occurred
```

## Mocking & Dependencies

**Pattern:**
k3d uses concrete implementations rather than extensive mocking. Tests pass real struct instances:

**Runtime Interface:**
Functions accept `runtime` parameter implementing the `Runtime` interface:
```go
func TestValidateClusterConfig(t *testing.T) {
    // Pass concrete docker runtime implementation
    err := ValidateClusterConfig(context.Background(), runtimes.Docker, cfg.(conf.ClusterConfig))
}

func TestTranslateNodeToContainer(t *testing.T) {
    // Create concrete Node struct for testing
    inputNode := &k3d.Node{...}
    // Function under test translates to Docker container config
    result := NodeInDocker{...}
}
```

**Viper Configuration:**
Tests that need config file loading use viper:
```go
vip := viper.New()
vip.SetConfigFile(cfgFile)
_ = vip.ReadInConfig()
cfg, err := FromViper(vip)
```

**What NOT to Mock:**
- Container runtime operations (tests use real Docker implementation)
- Configuration transformations (use real viper)
- Data structures (build real instances)

**What to Mock (if needed):**
- External API calls (not typical in this codebase)
- System-level operations (time, OS calls)

## Test Organization by Package

**`pkg/config/` Tests:**
- Configuration parsing and validation
- YAML fixture-based testing
- Transformation logic between config versions
- Fixtures: `test_assets/config_test_*.yaml`

**`pkg/util/` Tests:**
- Utility function testing (YAML splitting, encoding, etc.)
- Inline test data in table-driven tests
- No external fixtures typically needed

**`cmd/util/` Tests:**
- CLI utility testing (port parsing, filtering, etc.)
- Focused on parsing and validation logic
- Example: `ports_test.go` tests port exposure spec parsing

**`pkg/client/` Tests:**
- Client library function testing
- Registry configuration generation
- Inline struct construction for test data

**`pkg/runtimes/docker/` Tests:**
- Runtime-specific translation logic
- Docker container configuration translation
- Input: k3d Node structs
- Output: Docker container configuration structs

## Coverage

**Requirements:** No enforced coverage target

**View Coverage:**
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## CI/CD Testing

**CI Test Commands:**
```bash
make ci-tests  # Runs: fmt check lint e2e
make ci-lint   # Runs: golangci-lint with timeout
```

**E2E Testing:**
- Separate test suite from unit tests
- Run via `make e2e` with environment variables
- Environment variables control E2E behavior:
  - `E2E_LOG_LEVEL` - Logging level for E2E tests
  - `E2E_INCLUDE` - Filter tests to include
  - `E2E_EXCLUDE` - Filter tests to exclude
  - `E2E_KEEP` - Keep test artifacts for debugging
  - `E2E_PARALLEL` - Parallel test execution
  - `E2E_K3S_VERSION` - K3s version for testing
  - `E2E_FAIL_FAST` - Stop on first failure
- E2E tests defined in `tests/dind.sh`

## Test Dependencies

**Direct Dependencies:**
- `github.com/stretchr/testify` v1.10.0 - Assertion library
- `gotest.tools/assert` - Table-driven test assertions
- `github.com/go-test/deep` - Deep equality for complex types
- `github.com/spf13/viper` - Configuration loading in tests

**Runtime Dependencies (not mocked):**
- Docker runtime (for runtime-level tests)
- Configuration file system access

## Common Test Utilities

**Fixture Loading:**
```go
cfgFile := "./test_assets/config_test_simple.yaml"
vip := viper.New()
vip.SetConfigFile(cfgFile)
_ = vip.ReadInConfig()
```

**Struct Comparison:**
```go
// Using go-test/deep for complex comparisons
if diff := deep.Equal(expected, actual); diff != nil {
    t.Error(diff)
}
```

**String Assertions in Tests:**
```go
// Trim and compare YAML strings
assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))

// Substring checks
assert.Assert(t, strings.Contains(v, k3s.K3sPathStorage))
assert.Assert(t, strings.HasPrefix(v, "/tmp/testexpansion"))
```

---

*Testing analysis: 2026-02-05*
