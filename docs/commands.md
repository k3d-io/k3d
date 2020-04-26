# Command Tree

```bash
k3d
  --runtime  # choose the container runtime (default: docker)
  --verbose  # enable verbose (debug) logging (default: false)
  create
    cluster [CLUSTERNAME]  # default cluster name is 'k3s-default'
      --api-port
      --datastore-endpoint
      --image
      --k3s-agent-arg
      --k3s-server-arg
      --masters
      --network
      --no-image-volume
      --port
      --secret
      --timeout
      --update-kubeconfig
      --volume
      --wait
      --workers
    node NODENAME
  delete
    cluster CLUSTERNAME
    node
  start
    cluster CLUSTERNAME
    node
  stop
    cluster CLUSTERNAME
    node
  get
    cluster CLUSTERNAME
    node
    kubeconfig CLUSTERNAME
  load
  completion
  version
  help
```
