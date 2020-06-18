# Handling Kubeconfigs

By default, k3d won't touch your kubeconfig without you telling it to do so.
To get a kubeconfig set up for you to connect to a k3d cluster, you can go different ways.

??? question "What is the default kubeconfig?"
    We determine the path of the used or default kubeconfig in two ways:

    1. Using the `KUBECONFIG` environment variable, if it specifies *exactly one* file
    2. Using the default path (e.g. on Linux it's `#!bash $HOME/.kube/config`)

## Getting the kubeconfig for a newly created cluster

1. Create a new kubeconfig file **after** cluster creation
    - `#!bash k3d get kubeconfig mycluster`
      - *Note:* this will create (or update) the file `$HOME/.k3d/kubeconfig-mycluster.yaml`
      - *Tip:* pipe this: `export KUBECONFIG?$(k3d get kubeconfig mycluster)`
2. Update your default kubeconfig **upon** cluster creation
    - `#!bash k3d create cluster mycluster --update-kubeconfig`
        - *Note:* this won't switch the current-context (append `--switch` to do so)
3. Update your default kubeconfig **after** cluster creation
    - `#!bash k3d get kubeconfig mycluster --default`
        - *Note:* this won't switch the current-context (append `--switch` to do so)
4. Update a different kubeconfig **after** cluster creation
    - `#!bash k3d get kubeconfig mycluster --output some/other/file.yaml`
        - *Note:* this won't switch the current-context
    - The file will be created if it doesn't exist

!!! info "Switching the current context"
    None of the above options switch the current-context by default.
    This is intended to be least intrusive, since the current-context has a global effect.
    You can switch the current-context directly with the `get kubeconfig` command by adding the `--switch` flag.

## Removing cluster details from the kubeconfig

`#!bash k3d delete cluster mycluster` will always remove the details for `mycluster` from the default kubeconfig.
It will also delete the respective kubeconfig file in `$HOME/.k3d/` if it exists.

## Handling multiple clusters

`k3d get kubeconfig` let's you specify one or more clusters via arguments _or_ all via `--all`.
All kubeconfigs will then be merged into a single file if `--default` or `--output` is specified.
If none of those two flags was specified, a new file will be created per cluster and the merged path (e.g. `$HOME/.k3d/kubeconfig-cluster1.yaml:$HOME/.k3d/cluster2.yaml`) will be returned.
Note, that with multiple cluster specified, the `--switch` flag will change the current context to the cluster which was last in the list.
