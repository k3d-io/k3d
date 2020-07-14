# Defaults

- multiple server nodes
    - by default, when `--server` > 1 and no `--datastore-x` option is set, the first server node (server-0) will be the initializing server node
        - the initializing server node will have the `--cluster-init` flag appended
        - all other server nodes will refer to the initializing server node via `--server https://<init-node>:6443`
- API-Ports
    - by default, we don't expose any API-Port (no host port mapping)
- kubeconfig
    - if `--[update|merge]-default-kubeconfig` is set, we use the default loading rules to get the default kubeconfig:
        - First: kubeconfig specified via the KUBECONFIG environment variable (error out if multiple are specified)
        - Second: default kubeconfig in home directory (e.g. `$HOME/.kube/config`)
