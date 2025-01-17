# Using Podman instead of Docker

Podman has an [Docker API compatibility layer](https://podman.io/blogs/2020/06/29/podman-v2-announce.html#restful-api). k3d uses the Docker API and is compatible with Podman v4 and higher.

!!! important "Podman support is experimental"
    k3d is not guaranteed to work with Podman. If you find a bug, do help by [filing an issue](https://github.com/k3d-io/k3d/issues/new?labels=bug&template=bug_report.md&title=%5BBUG%5D+Podman)

Tested with podman version:
```bash
Client:       Podman Engine
Version:      4.3.1
API Version:  4.3.1
```

## Using Podman

Ensure the Podman system socket is available:

```bash
sudo systemctl enable --now podman.socket
# or to start the socket daemonless
# sudo podman system service --time=0 &
```

Disable timeout for podman service:<br>
See the [podman-system-service (1)](https://docs.podman.io/en/latest/markdown/podman-system-service.1.html) man page for more information.
```bash
mkdir -p /etc/containers/containers.conf.d
echo 'service_timeout=0' > /etc/containers/containers.conf.d/timeout.conf
```

To point k3d at the right Docker socket, create a symbolic link:

```bash
sudo ln -s /run/podman/podman.sock /var/run/docker.sock
# or install your system podman-docker if available
sudo k3d cluster create
```

Alternatively, set `DOCKER_HOST` when running k3d:

```bash
export DOCKER_HOST=unix:///run/podman/podman.sock
export DOCKER_SOCK=/run/podman/podman.sock
sudo --preserve-env=DOCKER_HOST --preserve-env=DOCKER_SOCK k3d cluster create
```

### Using rootless Podman

Ensure the Podman user socket is available:

```bash
systemctl --user enable --now podman.socket
# or podman system service --time=0 &
```

Set `DOCKER_HOST` when running k3d:

```bash
XDG_RUNTIME_DIR=${XDG_RUNTIME_DIR:-/run/user/$(id -u)}
export DOCKER_HOST=unix://$XDG_RUNTIME_DIR/podman/podman.sock
export DOCKER_SOCK=$XDG_RUNTIME_DIR/podman/podman.sock
k3d cluster create
```

#### Using cgroup (v2)

By default, a non-root user can only get memory controller and pids controller to be delegated.

To run properly we need to enable CPU, CPUSET, and I/O delegation

!!! note "Make sure you're running cgroup v2"
    If `/sys/fs/cgroup/cgroup.controllers` is present on your system, you are using v2, otherwise you are using v1.

```bash
mkdir -p /etc/systemd/system/user@.service.d
cat > /etc/systemd/system/user@.service.d/delegate.conf <<EOF
[Service]
Delegate=cpu cpuset io memory pids
EOF
systemctl daemon-reload
```

Reference: [https://rootlesscontaine.rs/getting-started/common/cgroup2/#enabling-cpu-cpuset-and-io-delegation](https://rootlesscontaine.rs/getting-started/common/cgroup2/#enabling-cpu-cpuset-and-io-delegation)

### Using remote Podman

[Start Podman on the remote host](https://github.com/containers/podman/blob/main/docs/tutorials/remote_client.md), and then set `DOCKER_HOST` when running k3d:

```
export DOCKER_HOST=ssh://username@hostname
export DOCKER_SOCK=/run/user/1000/podman/podman.sock
k3d cluster create
```

### macOS

Initialize a podman machine if not done already

```
podman machine init
```

Or start an already existing podman machine

```
podman machine start
```

Grab connection details 

```
podman system connection ls
Name                         URI                                                         Identity                                      Default
podman-machine-default       ssh://core@localhost:53685/run/user/501/podman/podman.sock  /Users/myusername/.ssh/podman-machine-default  true
podman-machine-default-root  ssh://root@localhost:53685/run/podman/podman.sock           /Users/myusername/.ssh/podman-machine-default  false
```

Edit your OpenSSH config file to specify the IdentityFile

```
vim ~/.ssh/config

Host localhost
	IdentityFile /Users/myusername/.ssh/podman-machine-default
```

#### Rootless mode

Delegate the `cpuset` cgroup controller to the user's systemd slice, export the docker environment variables referenced above for the non-root connection, and create the cluster:

```bash
podman machine ssh bash -e <<EOF
  printf '[Service]\nDelegate=cpuset\n' | sudo tee /etc/systemd/system/user@.service.d/k3d.conf
  sudo systemctl daemon-reload
  sudo systemctl restart "user@\${UID}"
EOF

export DOCKER_HOST=ssh://core@localhost:53685
export DOCKER_SOCKET=/run/user/501/podman/podman.sock
k3d cluster create --k3s-arg '--kubelet-arg=feature-gates=KubeletInUserNamespace=true@server:*'
```

#### Rootful mode

Export the docker environment variables referenced above for the root connection and create the cluster:

```bash
export DOCKER_HOST=ssh://root@localhost:53685
export DOCKER_SOCK=/run/podman/podman.sock
k3d cluster create
```

### Podman network

The default `podman` network has dns disabled. To allow k3d cluster nodes to communicate with dns a new network must be created.
```bash
podman network create k3d
podman network inspect k3d -f '{{ .DNSEnabled }}'
true
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

!!! note "Missing cpuset cgroup controller"
    If you experince an error regarding missing cpuset cgroup controller, ensure the user unit `xdg-document-portal.service` is disabled by running `systemctl --user stop xdg-document-portal.service`. See [this issue](https://github.com/systemd/systemd/issues/18293#issuecomment-831397578)

