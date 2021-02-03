# Changelog

## v4.1.0

### Important

- Using Viper brings us lots of nice features, but also one problem:
  - We had to switch StringArray flags to StringSlice flags, which
    - allow to use multiple flag values comma-separated in a single flag, but also
    - split flag values that contain a comma into separate parts (and we cannot handle issues that arise due to this)
      - so if you rely on commas in your flag values (e.g. for `--env X=a,b,c`), please consider filing an issue or supporting <https://github.com/spf13/viper/issues/246> and <https://github.com/spf13/viper/pull/398>
      - `--env X=a,b,c` would be treated the same as `--env X=a`, `--env b`, `--env c`

### Features & Enhancements

- use [viper](https://github.com/spf13/viper) for configuration management
  - takes over the job of properly fetching and merging config options from
    - CLI arguments/flags
    - environment variables
    - config file
  - this also fixes some issues with using the config file (like cobra defaults overriding config file values)
- add JSON-Schema validation for the `Simple` config file schema
- new config version `k3d.io/v1alpha2` (some naming changes)
  - `exposeAPI` -> `kubeAPI`
  - `options.k3d.noRollback` -> `options.k3d.disableRollback`
  - `options.k3d.prepDisableHostIPInjection` -> `options.k3d.disableHostIPInjection`

### Misc

- tests/e2e: add config override test

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
