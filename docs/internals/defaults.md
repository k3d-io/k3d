# Defaults

## Multiple server nodes

- by default, when `--server` > 1 and no `--datastore-x` option is set, the first server node (server-0) will be the initializing server node
  - the initializing server node will have the `--cluster-init` flag appended
  - all other server nodes will refer to the initializing server node via `--server https://<init-node>:6443`

## API-Ports

- by default, we expose the API-Port (`6443`) by forwarding traffic from the default server loadbalancer (nginx container) to the server node(s)
- port `6443` of the loadbalancer is then mapped to a specific (`--api-port` flag) or a random (default) port on the host system

## Kubeconfig

- if `--kubeconfig-update-default` is set, we use the default loading rules to get the default kubeconfig:
  - First: kubeconfig specified via the KUBECONFIG environment variable (error out if multiple are specified)
  - Second: default kubeconfig in home directory (e.g. `$HOME/.kube/config`)

## Networking

- [by default, k3d creates a new (docker) network for every cluster](./networking)
