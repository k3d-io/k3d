# Changelog

## v4.0.0

### Breaking Changes

#### Module

**If you're using k3d as a Go module, please have a look into the code to see all the changes!**

- We're open for chats via Slack or GitHub discussions

- Module is now on `github.com/rancher/k3d/v4` due to lots of breaking changes
- `pkg/cluster` is now `pkg/client`
- `ClusterCreate` and `NodeCreate` don't start the entities (containers) anymore
  - `ClusterRun` and `NodeRun` orchestrate the new Create and Start functionality
- New config flow: CLIConfig (SimpleConfig) -> ClusterConfig -> Cluster + Opts

#### CLI

- Some flags changed to also use `noun-action` syntax
  - e.g. `--switch-context --update-default-kubeconfig` -> `--kubeconfig-switch-context --kubeconfig-update-default`
  - this eases grouping and visibility

### Changes

#### Features

- Registry Support
  - k3d-managed registry like we had it in k3d v1.x
  - Option 1: default settings, paired with cluster creation
    - `k3d cluster create --registry-create` -> New registry for that cluster
    - `k3d cluster create --registry-use` -> Re-use existing registry
  - Option 2: customized, managed stand-alone
    - `k3d registry [create/start/stop/delete]`
    - Check the documentation, help text and tutorials for more details
  - Communicate managed registry using the LocalRegistryHostingV1 spec from [KEP-1755](https://github.com/kubernetes/enhancements/blob/0d69f7cea6fbe73a7d70fab569c6898f5ccb7be0/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry/README.md)
    - interesting especially for tools that reload images, like Tilt or Skaffold

- Config File Support
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

- [WIP] Support for Lifecycle Hooks
  - Run any executable at specific stages during the cluster and node lifecycles
    - e.g. we modify the `registries.yaml` in the `preStart` stage of nodes
    - Guides will follow

#### Misc

- Now building with Go 1.15
  - same for the k3d-tools code
- updated dependencies (including Docker v20.10)
- tests/e2e: add E2E_INCLUDE and rename E2E_SKIP to E2E_EXCLUDE
