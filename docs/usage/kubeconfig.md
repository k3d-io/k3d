# Handling Kubeconfigs

By default, k3d won't touch your kubeconfig without you telling it to do so.
To get a kubeconfig set up for you to connect to a k3d cluster, you can go different ways.

??? question "What is the default kubeconfig?"
    We determine the path of the used or default kubeconfig in two ways:

    1. Using the `KUBECONFIG` environment variable, if it specifies *exactly one* file
    2. Using the default path (e.g. on Linux it's `#!bash $HOME/.kube/config`)

## Getting the kubeconfig for a newly created cluster

1. Update your default kubeconfig **upon** cluster creation
    - `#!bash k3d cluster create mycluster --update-kubeconfig`
        - *Note:* this won't switch the current-context
2. Update your default kubeconfig **after** cluster creation
    - `#!bash k3d kubeconfig merge mycluster`
        - *Note:* this won't switch the current-context
3. Update a different kubeconfig **after** cluster creation
    - `#!bash k3d kubeconfig merge mycluster --output some/other/file.yaml`
        - *Note:* this won't switch the current-context
    - The file will be created if it doesn't exist

!!! info "Switching the current context"
    None of the above options switch the current-context.
    This is intended to be least intrusive, since the current-context has a global effect.
    You can switch the current-context directly with the `kubeconfig merge` command by adding the `--switch` flag.

## Removing cluster details from the kubeconfig

`#!bash k3d cluster delete mycluster` will always remove the details for `mycluster` from the default kubeconfig.

## Handling multiple clusters

`k3s kubeconfig merge` let's you specify one or more clusters via arguments _or_ all via `--all`.
All kubeconfigs will then be merged into a single file, which is either the default kubeconfig or the kubeconfig specified via `--output FILE`.
Note, that with multiple cluster specified, the `--switch` flag will change the current context to the cluster which was last in the list.
