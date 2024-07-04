# FAQ

## Issues with BTRFS

- As [@jaredallard](https://github.com/jaredallard) [pointed out](https://github.com/k3d-io/k3d/pull/48), people running `k3d` on a system with **btrfs**, may need to mount `/dev/mapper` into the nodes for the setup to work.
  - This will do: `#!bash k3d cluster create CLUSTER_NAME -v /dev/mapper:/dev/mapper`

## Issues with ZFS

- k3s currently has [no support for ZFS](https://github.com/rancher/k3s/issues/66) and thus, creating multi-server setups (e.g. `#!bash k3d cluster create multiserver --servers 3`) fails, because the initializing server node (server flag `--cluster-init`) errors out with the following log:

  ```bash
  starting kubernetes: preparing server: start cluster and https: raft_init(): io: create I/O capabilities probe file: posix_allocate: operation not supported on socket
  ```

  - This issue can be worked around by providing docker with a different filesystem (that's also better for docker-in-docker stuff).
  - A possible solution can be found here: [https://github.com/rancher/k3s/issues/1688#issuecomment-619570374](https://github.com/rancher/k3s/issues/1688#issuecomment-619570374)

## Pods evicted due to lack of disk space

- Pods go to evicted state after doing X
  - Related issues: [#133 - Pods evicted due to `NodeHasDiskPressure`](https://github.com/k3d-io/k3d/issues/133) (collection of #119 and #130)
  - Background: somehow docker runs out of space for the k3d node containers, which triggers a hard eviction in the kubelet
  - Possible [fix/workaround by @zer0def](https://github.com/k3d-io/k3d/issues/133#issuecomment-549065666):
    - cleanup your host file system: Yes, your host file system may actually be quite packed, triggering the eviction threshold.
      - on large disks, you may still have quite a few GB leftover, which is more than enough. In that case, lower the threshold as per below.
    - use a docker storage driver which cleans up properly (e.g. overlay2)
    - clean up or expand docker root filesystem
    - change the kubelet's eviction thresholds upon cluster creation:

      ```bash
      k3d cluster create \
        --k3s-arg '--kubelet-arg=eviction-hard=imagefs.available<1%,nodefs.available<1%@agent:*' \
        --k3s-arg '--kubelet-arg=eviction-minimum-reclaim=imagefs.available=1%,nodefs.available=1%@agent:*'
      ```

## Passing additional arguments/flags to k3s (and on to e.g. the kube-apiserver)

- The Problem: Passing a feature flag to the Kubernetes API Server running inside k3s.
- Example: you want to enable the EphemeralContainers feature flag in Kubernetes
- Solution:

  ```bash
    k3d cluster create \
    --k3s-arg '--kube-apiserver-arg=feature-gates=EphemeralContainers=true@server:*' \
    --k3s-arg '--kube-scheduler-arg=feature-gates=EphemeralContainers=true@server:*' \
    --k3s-arg '--kubelet-arg=feature-gates=EphemeralContainers=true@agent:*'
  ```

  - **Note**: Be aware of where the flags require dashes (`--`) and where not.
    - the k3s flag (`--kube-apiserver-arg`) has the dashes
    - the kube-apiserver flag `feature-gates` doesn't have them (k3s adds them internally)

- Second example:

  ```bash
  k3d cluster create k3d-one \
    --k3s-arg "--cluster-cidr=10.118.0.0/17@server:*" \
    --k3s-arg "--service-cidr=10.118.128.0/17@server:*" \
    --k3s-arg "--disable=servicelb@server:*" \
    --k3s-arg "--disable=traefik@server:*" \
    --verbose
  ```

  - **Note**: There are many ways to use the `"` and `'` quotes, just be aware, that sometimes shells also try to interpret/interpolate parts of the commands

## How to access services (like a database) running on my Docker Host Machine

- As of version v3.1.0, we're injecting the `host.k3d.internal` entry into the k3d containers (k3s nodes) and into the CoreDNS ConfigMap, enabling you to access your host system by referring to it as `host.k3d.internal`

## Running behind a corporate proxy

Running k3d behind a corporate proxy can lead to some issues with k3d that have already been reported in more than one issue.  
Some can be fixed by passing the `HTTP_PROXY` environment variables to k3d, some have to be fixed in docker's `daemon.json` file and some are as easy as adding a volume mount.

## Pods fail to start: `x509: certificate signed by unknown authority`

- Example Error Message:

  ```bash
  Failed to create pod sandbox: rpc error: code = Unknown desc = failed to get sandbox image "docker.io/rancher/pause:3.1": failed to pull image "docker.io/rancher/pause:3.1": failed to pull and unpack image "docker.io/rancher/pause:3.1": failed to resolve reference "docker.io/rancher/pause:3.1": failed to do request: Head https://registry-1.docker.io/v2/rancher/pause/manifests/3.1: x509: certificate signed by unknown authority
  ```

- Problem: inside the container, the certificate of the corporate proxy cannot be validated
- Possible Solution: Mounting the CA Certificate from your host into the node containers at start time via `k3d cluster create --volume /path/to/your/certs.crt:/etc/ssl/certs/yourcert.crt`
- Issue: [k3d-io/k3d#535](https://github.com/k3d-io/k3d/discussions/535#discussioncomment-474982)

## Spurious PID entries in `/proc` after deleting `k3d` cluster with shared mounts

- When you perform cluster create and deletion operations multiple times with **same cluster name** and **shared volume mounts**, it was observed that `grep k3d /proc/*/mountinfo` shows many spurious entries
- Problem: Due to above, at times you'll see `no space left on device: unknown` when a pod is scheduled to the nodes
- If you observe anything of above sort you can check for inaccessible file systems and unmount them by using below command (note: please remove `xargs umount -l` and check for the diff o/p first)
- `diff <(df -ha | grep pods | awk '{print $NF}') <(df -h | grep pods | awk '{print $NF}') | awk '{print $2}' | xargs umount -l`
- As per the conversation on [k3d-io/k3d#594](https://github.com/k3d-io/k3d/issues/594#issuecomment-837900646) above issue wasn't reported/known earlier and so there are high chances that it's not universal.

## [SOLVED] Nodes fail to start or get stuck in `NotReady` state with log `nf_conntrack_max: permission denied`

### Problem

- When: This happens when running k3d on a Linux system with a kernel version >= 5.12.2 (and others like >= 5.11.19) when creating a new cluster
  - the node(s) stop or get stuck with a log line like this: `<TIMESTAMP>  F0516 05:05:31.782902       7 server.go:495] open /proc/sys/net/netfilter/nf_conntrack_max: permission denied`
- Why: The issue was introduced by a change in the Linux kernel ([Changelog 5.12.2](https://cdn.kernel.org/pub/linux/kernel/v5.x/ChangeLog-5.12.2): [Commit](https://github.com/torvalds/linux/commit/671c54ea8c7ff47bd88444f3fffb65bf9799ce43)), that changed the netfilter_conntrack behavior in a way that `kube-proxy` is not able to set the `nf_conntrack_max` value anymore

### Workaround

- Workaround: as a workaround, we can tell `kube-proxy` to not even try to set this value:

  ```bash
  k3d cluster create \
    --k3s-arg "--kube-proxy-arg=conntrack-max-per-core=0@server:*" \
    --k3s-arg "--kube-proxy-arg=conntrack-max-per-core=0@agent:*" \
    --image rancher/k3s:v1.20.6-k3s
  ```

### Fix

- **Note**: k3d v4.4.5 already uses rancher/k3s:v1.21.1-k3s1 as the new default k3s image, so no workarounds needed there!

This is going to be fixed "upstream" in k3s itself in [rancher/k3s#3337](https://github.com/k3s-io/k3s/pull/3337) and backported to k3s versions as low as v1.18.

- **The fix was released and backported in k3s, so you don't need to use the workaround when using one of the following k3s versions (or later ones)**
  - v1.18.19-k3s1 ([rancher/k3s#3344](https://github.com/k3s-io/k3s/pull/3344))
  - v1.19.11-k3s1 ([rancher/k3s#3343](https://github.com/k3s-io/k3s/pull/3343))
  - v1.20.7-k3s1 ([rancher/k3s#3342](https://github.com/k3s-io/k3s/pull/3342))
  - v1.21.1-k3s1 ([rancher/k3s#3341](https://github.com/k3s-io/k3s/pull/3341)))
- Issue Reference: [rancher/k3s#607](https://github.com/k3d-io/k3d/issues/607)

## DockerHub Pull Rate Limit

### Problem

You're deploying something to the cluster using an image from DockerHub and the image fails to be pulled, with a `429` response code and a message saying `You have reached your pull rate limit. You may increase the limit by authenticating and upgrading`.

### Cause

This is caused by DockerHub's pull rate limit (see <https://docs.docker.com/docker-hub/download-rate-limit/>), which limits pulls from unauthenticated/anonymous users to 100 pulls per hour and for authenticated users (not paying customers) to 200 pulls per hour (as of the time of writing).

### Solution

a) use images from a private registry, e.g. configured as a pull-through cache for DockerHub  
b) use a different public registry without such limitations, if the same image is stored there  
c) authenticate containerd inside k3s/k3d to use your DockerHub user  

#### (c) Authenticate Containerd against DockerHub

1. Create a registry configuration file for containerd:

  ```yaml
  # saved as e.g. $HOME/registries.yaml
  configs:
    "docker.io":
      auth:
        username: "$USERNAME"
        password: "$PASSWORD"
  ```

2. Create a k3d cluster using that config:

  ```bash
  k3d cluster create --registry-config $HOME/registries.yaml
  ```

3. Profit. That's it. In the test for this, we pulled the same image 120 times in a row (confirmed, that pull numbers went up), without being rate limited (as a non-paying, normal user)

## Longhorn in k3d

### Problem

Longhorn is not working when deployed in a K3s cluster spawned with k3d.

### Cause

The container image of K3s is quite limited and doesn't contain the necessary libraries.  Also, additional volume mounts and more would be required to get Longhorn up and running properly.  
So basically Longhorn does rely too much on the host OS to work properly in the dockerized environment without quite some modifications.

### Solution

There are a few ways one can build a working image to use with k3d.  
See <https://github.com/k3d-io/k3d/discussions/478> for more info.
