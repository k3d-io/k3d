# FAQ / Nice to know

- As [@jaredallard](https://github.com/jaredallard) [pointed out](https://github.com/rancher/k3d/pull/48), people running `k3d` on a system with **btrfs**, may need to mount `/dev/mapper` into the nodes for the setup to work.
  - This will do: `k3d create -v /dev/mapper:/dev/mapper`
  - An additional solution proposed by [@zer0def](https://github.com/zer0def) can be found in the [examples section](examples.md) (_Running on filesystems k3s doesn't like (btrfs, tmpfs, …)_)
