# Using Podman instead of Docker

Podman has an [Docker API compatibility layer](https://podman.io/blogs/2020/06/29/podman-v2-announce.html#restful-api). k3d uses the Docker API and is compatible with Podman v4 and higher.

!!! important "Podman support is experimental"
    k3d is not guaranteed to work with Podman. If you find a bug, do help by [filing an issue](https://github.com/k3d-io/k3d/issues/new?labels=bug&template=bug_report.md&title=%5BBUG%5D+Podman)

## Using Podman

Ensure the Podman system socket is available:

```bash
sudo systemctl enable --now podman.socket
# or sudo podman system service --time=0
```

To point k3d at the right Docker socket, create a symbolic link:

```bash
ln -s /run/podman/podman.sock /var/run/docker.sock
# or install your system podman-docker if available
sudo k3d cluster create
```

Alternatively, set DOCKER_HOST when running k3d:

```bash
export DOCKER_HOST=unix:///run/podman/podman.sock
sudo --preserve-env=DOCKER_HOST k3d cluster create
```

### Using rootless Podman

Ensure the Podman user socket is available:

```bash
systemctl --user enable --now podman.socket
# or podman system service --time=0
```

Set DOCKER_HOST when running k3d:

```bash
XDG_RUNTIME_DIR=${XDG_RUNTIME_DIR:-/run/user/$(id -u)}
export DOCKER_HOST=unix://$XDG_RUNTIME_DIR/podman/podman.sock
k3d cluster create
```

## Creating local registries

Because Podman does not have a default "bridge" network, you have to specify a network using the `--default-network` flag when creating a local registry:

```bash
k3d registry create --default-network podman mycluster-registry
```

To use this registry with a cluster, pass the `--registry-use` flag:

```bash
k3d cluster create --registry-use mycluster-registry mycluster
```

!!! note "Incompatibility with `--registry-create`"
    Because `--registry-create` assumes the default network to be "bridge", avoid `--registry-create` when using Podman. Instead, always create a registry before creating a cluster.
