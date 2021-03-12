# FAQ / Nice to know

## Issues with BTRFS

- As [@jaredallard](https://github.com/jaredallard) [pointed out](https://github.com/rancher/k3d/pull/48), people running `k3d` on a system with **btrfs**, may need to mount `/dev/mapper` into the nodes for the setup to work.
  - This will do: `k3d cluster create CLUSTER_NAME -v /dev/mapper:/dev/mapper`

## Issues with ZFS

- k3s currently has [no support for ZFS](ttps://github.com/rancher/k3s/issues/66) and thus, creating multi-server setups (e.g. `k3d cluster create multiserver --servers 3`) fails, because the initializing server node (server flag `--cluster-init`) errors out with the following log:

  ```bash
  starting kubernetes: preparing server: start cluster and https: raft_init(): io: create I/O capabilities probe file: posix_allocate: operation not supported on socket
  ```

  - This issue can be worked around by providing docker with a different filesystem (that's also better for docker-in-docker stuff).
  - A possible solution can be found here: [https://github.com/rancher/k3s/issues/1688#issuecomment-619570374](https://github.com/rancher/k3s/issues/1688#issuecomment-619570374)

## Pods evicted due to lack of disk space

- Pods go to evicted state after doing X
  - Related issues: [#133 - Pods evicted due to `NodeHasDiskPressure`](https://github.com/rancher/k3d/issues/133) (collection of #119 and #130)
  - Background: somehow docker runs out of space for the k3d node containers, which triggers a hard eviction in the kubelet
  - Possible [fix/workaround by @zer0def](https://github.com/rancher/k3d/issues/133#issuecomment-549065666):
    - use a docker storage driver which cleans up properly (e.g. overlay2)
    - clean up or expand docker root filesystem
    - change the kubelet's eviction thresholds upon cluster creation: `k3d cluster create --k3s-agent-arg '--kubelet-arg=eviction-hard=imagefs.available<1%,nodefs.available<1%' --k3s-agent-arg '--kubelet-arg=eviction-minimum-reclaim=imagefs.available=1%,nodefs.available=1%'`

## Restarting a multi-server cluster or the initializing server node fails

- What you do: You create a cluster with more than one server node and later, you either stop `server-0` or stop/start the whole cluster
- What fails: After the restart, you cannot connect to the cluster anymore and `kubectl` will give you a lot of errors
- What causes this issue: it's a [known issue with dqlite in `k3s`](https://github.com/rancher/k3s/issues/1391) which doesn't allow the initializing server node to go down
- What's the solution: Hopefully, this will be solved by the planned [replacement of dqlite with embedded etcd in k3s](https://github.com/rancher/k3s/pull/1770)
- Related issues: [#262](https://github.com/rancher/k3d/issues/262)

## Passing additional arguments/flags to k3s (and on to e.g. the kube-apiserver)

- The Problem: Passing a feature flag to the Kubernetes API Server running inside k3s.
- Example: you want to enable the EphemeralContainers feature flag in Kubernetes
- Solution: `#!bash k3d cluster create --k3s-server-arg '--kube-apiserver-arg=feature-gates=EphemeralContainers=true'`
  - Note: Be aware of where the flags require dashes (`--`) and where not.
    - the k3s flag (`--kube-apiserver-arg`) has the dashes
    - the kube-apiserver flag `feature-gates` doesn't have them (k3s adds them internally)

- Second example: `#!bash k3d cluster create k3d-one --k3s-server-arg --cluster-cidr="10.118.0.0/17" --k3s-server-arg --service-cidr="10.118.128.0/17" --k3s-server-arg --disable=servicelb --k3s-server-arg --disable=traefik --verbose`
  - Note: There are many ways to use the `"` and `'` quotes, just be aware, that sometimes shells also try to interpret/interpolate parts of the commands

## How to access services (like a database) running on my Docker Host Machine

- As of version v3.1.0, we're injecting the `host.k3d.internal` entry into the k3d containers (k3s nodes) and into the CoreDNS ConfigMap, enabling you to access your host system by referring to it as `host.k3d.internal`

## Running behind a corporate proxy

Running k3d behind a corporate proxy can lead to some issues with k3d that have already been reported in more than one issue.
Some can be fixed by passing the `HTTP_PROXY` environment variables to k3d, some have to be fixed in docker's `daemon.json` file and some are as easy as adding a volume mount.

### Pods fail to start: `x509: certificate signed by unknown authority`

- Example Error Message: `Failed to create pod sandbox: rpc error: code = Unknown desc = failed to get sandbox image "docker.io/rancher/pause:3.1": failed to pull image "docker.io/rancher/pause:3.1": failed to pull and unpack image "docker.io/rancher/pause:3.1": failed to resolve reference "docker.io/rancher/pause:3.1": failed to do request: Head https://registry-1.docker.io/v2/rancher/pause/manifests/3.1: x509: certificate signed by unknown authority`
- Problem: inside the container, the certificate of the corporate proxy cannot be validated
- Possible Solution: Mounting the CA Certificate from your host into the node containers at start time via `k3d cluster create --volume /path/to/your/certs.crt:/etc/ssl/certs/yourcert.crt`
- Issue: [rancher/k3d#535](https://github.com/rancher/k3d/discussions/535#discussioncomment-474982)
