# Defaults

## k3d reserved settings

When you create a K3s cluster in Docker using k3d, we make use of some K3s configuration options, making them "reserved" for k3d.
This means, that overriding those options with your own may break the cluster setup.

### Environment Variables

The following K3s environment variables are used to configure the cluster:

| Variable | K3d Default | Configurable? |
|----------|-------------|---------------|
| `K3S_URL`| `https://$CLUSTERNAME-server-0:6443` | no |
| `K3S_TOKEN`| random | yes (`--token`) |
| `K3S_KUBECONFIG_OUTPUT`| `/output/kubeconfig.yaml` | no |

## k3d Loadbalancer

By default, k3d creates an Nginx loadbalancer alongside the clusters it creates to handle the port-forwarding.
The loadbalancer can partly be configured using k3d-defined settings.

| Nginx setting | k3d default | k3d setting |
|-------------|-------------|-------------|
| `proxy_timeout` (default for all server stanzas) | `600` (s) | `settings.defaultProxyTimeout` |  |
|`worker_connections` | `1024` | `settings.workerConnections` |

### Overrides

- Example via CLI: `k3d cluster create --lb-config-override settings.defaultProxyTimeout=900`
- Example via Config File:

  ```yaml
  # ... truncated ...
  k3d:
    loadbalancer:
      configOverrides:
        - settings.workerConnections=2048
  ```

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
