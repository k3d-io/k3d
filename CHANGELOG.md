# Changelog

## v4.4.0

### Features / Enhancements

- Support for Memory Limits using e.g. `--servers-memory 1g` or `--agents-memory 1.5g` (#494, @konradmalik)
  - enabled by providing fake `meminfo` files

### Fixes

- fix absolute paths in volume mounts on Windows (#510, @markrexwinkel)

### Documentation

- clarify registry names in docs and help text
- add usage section about config file (#534)
- add FAQ entry on certificate error when running behind corporate proxy
- add MacPorts install instructions (#539, @herbygillot)
- Heal Shruggie: Replace amputated arm (#540, @claycooper)

## v4.3.0

### Features / Enhancements

- Use Go 1.16
  - update dependencies, including kubernetes, docker, containerd and more
  - add `darwin/arm64` (Apple Silicon, M1) build target (#530)
  - use the new `//go:embed` feature to directly embed the jsonschema in the binary (#529)
- Add a status column to `k3d registry list` output (#496, @ebr)
- Allow non-prefixed (i.e. without `k3d-` prefix) user input when fetching resources (e.g. `k3d node get mycluster-server-0` would return successfully)

### Fixes

- Allow absolute paths for volumes on Windows (#510, @markrexwinkel)
- fix nil-pointer exception in case of non-existent IPAM network config
- Properly handle combinations of host/hostIP in kubeAPI settings reflected in the kubeconfig (#500, @fabricev)

### Misc

- docs: fix typo in stop command help text (#513, @searsaw)
- ci/ghaction: AUR (pre-)release now on Ubuntu 20.04 and latest archlinux image
- REMOVE incomplete and unused `containerd` runtime from codebase, as it was causing issues to build for windows and hasn't made any progress in quite some time now

## v4.2.0

### Features / Enhancements

- add processing step for cluster config, to configure it e.g. for hostnetwork mode (#477, @konradmalik)
- allow proxying UDP ports via the load balancer (#488, @k0da)

### Fixes

- fix usage of `DOCKER_HOST` env var for Kubeconfig server ref (trim port)
- fix error when trying to attach the same node (e.g. registry) to the same network twice (#486, @kuritka)
- fix Kube-API settings in configg file got overwritten (#490, @dtomasi)

### Misc

- add `k3d.version` label to created resources
- add Pull-Request template
- docs: add hint on minimal requirements for multi-server clusters (#481, @Filius-Patris)

## v4.1.1

### Fixes

- fix: `--k3s-server-arg` and `--k3s-agent-arg` didn't work (Viper StringArray incompatibility) (#482)

## v4.1.0

### Highlights

#### :scroll: Configuration Enhancements

- :snake: use [viper](https://github.com/spf13/viper) for configuration management
  - takes over the job of properly fetching and merging config options from
    - CLI arguments/flags
    - environment variables
    - config file
  - this also fixes some issues with using the config file (like cobra defaults overriding config file values)
- :heavy_check_mark: add JSON-Schema validation for the `Simple` config file schema
- :new: config version `k3d.io/v1alpha2` (some naming changes)
  - `exposeAPI` -> `kubeAPI`
  - `options.k3d.noRollback` -> `options.k3d.disableRollback`
  - `options.k3d.prepDisableHostIPInjection` -> `options.k3d.disableHostIPInjection`

#### :computer: Docker over SSH

- Support Docker over SSH (#324, @ekristen & @inercia)

### Features & Enhancements

- add root flag `--timestamps` to enable timestamped logs
- improved multi-server cluster support (#467)
  - log a warning, if one tries to create a cluster with only 2 nodes (no majority possible, no fault tolerance)
  - revamped cluster start procedure: init-node, sorted servers, agents, helpers
  - different log messages per role and start-place (that we wait for to consider a node to be ready)
  - module: `NodeStartOpts` now accept a `ReadyLogMessage` and `NodeState` now takes a `Started` timestamp string

### Fixes

- do not ignore `--no-hostip` flag and don't inject hostip if `--network=host` (#471, @konradmalik)
- fix: `--no-lb` ignored
- fix: print error cause when serverlb fails to start

### Misc

- tests/e2e: add config override test
- tests/e2e: add multi server start-stop cycle test
- tests/e2e: improved logs with stage and test details.
- builds&tests: use Docker 20.10 and BuildKit everywhere
- :memo: docs: add <https://github.com/AbsaOSS/k3d-action> (GitHub Action) as a related project (#476, @kuritka)

### Tested with

- E2E Tests ran with k3s versions
  - v1.17.17-k3s1 (see Known Issues below)
  - v1.18.15-k3s1 (see Known Issues below)
  - v1.19.7-k3s1
  - v1.20.2-k3s1

### Known Issues

- automatic multi-server cluster restarts tend to fail with k3s versions v1.17.x & v1.18.x and probably earlier versions (using dqlite)
- Using Viper brings us lots of nice features, but also one problem:
  - We had to switch StringArray flags to StringSlice flags, which
    - allow to use multiple flag values comma-separated in a single flag, but also
    - split flag values that contain a comma into separate parts (and we cannot handle issues that arise due to this)
      - so if you rely on commas in your flag values (e.g. for `--env X=a,b,c`), please consider filing an issue or supporting <https://github.com/spf13/viper/issues/246> and <https://github.com/spf13/viper/pull/398>
      - `--env X=a,b,c` would be treated the same as `--env X=a`, `--env b`, `--env c`

## v4.0.0

### Breaking Changes

#### Module

**If you're using k3d as a Go module, please have a look into the code to see all the changes!**

- We're open for chats via Slack or GitHub discussions

- Module is now on `github.com/rancher/k3d/v4` due to lots of breaking changes
- `pkg/cluster` is now `pkg/client`
- `ClusterCreate` and `NodeCreate` don't start the entities (containers) anymore
  - `ClusterRun` and `NodeRun` orchestrate the new Create and Start functionality
- `NodeDelete`/`ClusterDelete` now take an additional `NodeDeleteOpts`/`ClusterDeleteOpts` struct to toggle specific steps
- NodeSpec now features a list of networks (required for registries)
- New config flow: CLIConfig (SimpleConfig) -> ClusterConfig -> Cluster + Opts

#### CLI

- Some flags changed to also use `noun-action` syntax
  - e.g. `--switch-context --update-default-kubeconfig` -> `--kubeconfig-switch-context --kubeconfig-update-default`
  - this eases grouping and visibility

### Changes

#### Features

- **Registry Support**
  - k3d-managed registry like we had it in k3d v1.x
  - Option 1: default settings, paired with cluster creation
    - `k3d cluster create --registry-create` -> New registry for that cluster
    - `k3d cluster create --registry-use` -> Re-use existing registry
  - Option 2: customized, managed stand-alone
    - `k3d registry [create/start/stop/delete]`
    - Check the documentation, help text and tutorials for more details
  - Communicate managed registry using the LocalRegistryHostingV1 spec from [KEP-1755](https://github.com/kubernetes/enhancements/blob/0d69f7cea6fbe73a7d70fab569c6898f5ccb7be0/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry/README.md)
    - interesting especially for tools that reload images, like Tilt or Skaffold

- **Config File Support**
  - Put all your CLI-Arguments/Flags into a more readable config file and re-use it everywhere (keep it in your repo)
    - Note: this is not always a 1:1 matching in naming/syntax/semantics
  - `k3d cluster create --config myconfig.yaml`

    ```yaml
    apiVersion: k3d.io/v1alpha1
    kind: Simple
    name: mycluster
    servers: 3
    agents: 2
    ports:
      - port: 8080:80
        nodeFilters:
          - loadbalancer
    ```

  - Check out our test cases in [pkg/config/test_assets/](./pkg/config/test_assets/) for more config file examples
  - **Note**: The config file format (& feature) might still be a little rough around the edges and it's prone to change quickly until we hit a stable release of the config

- [WIP] Support for Lifecycle Hooks
  - Run any executable at specific stages during the cluster and node lifecycles
    - e.g. we modify the `registries.yaml` in the `preStart` stage of nodes
    - Guides will follow

- Print container creation time (#431, @inercia)
- add output formats for `cluster ls` and `node ls` (#439, @inercia)

#### Fixes

- import image: avoid nil pointer exception in specific cases
- cluster delete: properly handle node and network (#437)
- --port: fix bnil-pointer exception when exposing port on non-existent loadbalancer
- completion/zsh: source completion file

#### Misc

- Now building with Go 1.15
  - same for the k3d-tools code
- updated dependencies (including Docker v20.10)
- tests/e2e: add `E2E_INCLUDE` and rename `E2E_SKIP` to `E2E_EXCLUDE`
- tests/e2e: allow overriding the Helper Image Tag via `E2E_HELPER_IMAGE_TAG`
- docs: spell checking (#434, @jsoref)
- docs: add Chocolatey install option (#443, @erwinkersten)
