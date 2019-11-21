# FAQ / Nice to know

- As [@jaredallard](https://github.com/jaredallard) [pointed out](https://github.com/rancher/k3d/pull/48), people running `k3d` on a system with **btrfs**, may need to mount `/dev/mapper` into the nodes for the setup to work.
  - This will do: `k3d create -v /dev/mapper:/dev/mapper`
  - An additional solution proposed by [@zer0def](https://github.com/zer0def) can be found in the [examples section](examples.md) (_Running on filesystems k3s doesn't like (btrfs, tmpfs, …)_)

- Pods go to evicted state after doing X
  - Related issues: [#133 - Pods evicted due to `NodeHasDiskPressure`](https://github.com/rancher/k3d/issues/133) (collection of #119 and #130)
  - Background: somehow docker runs out of space for the k3d node containers, which triggers a hard eviction in the kubelet
  - Possible [fix/workaround by @zer0def](https://github.com/rancher/k3d/issues/133#issuecomment-549065666):
    - use a docker storage driver which cleans up properly (e.g. overlay2)
    - clean up or expand docker root filesystem
    - change the kubelet's eviction thresholds upon cluster creation: `k3d create --agent-arg '--eviction-hard=imagefs.available<1%,nodefs.available<1%' --agent-arg '--eviction-minimum-reclaim=imagefs.available=1%,nodefs.available=1%'`
