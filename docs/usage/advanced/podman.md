# Using Podman instead of Docker

Podman has an [Docker API compatibility layer](https://podman.io/blogs/2020/06/29/podman-v2-announce.html#restful-api).  
k3d uses the Docker API and is compatible with Podman v4 and higher.

!!! important "Podman support is experimental"
    k3d is not guaranteed to work with Podman. If you find a bug, do help by [filing an issue](https://github.com/k3d-io/k3d/issues/new?labels=bug&template=bug_report.md&title=%5BBUG%5D+Podman)

Tested with:
```
podman version 5.7.0
k3d version v5.8.3
```

## Basic Setup

Ensure the Podman system socket is available and exposed with the environment variable `DOCKER_HOST`.

```bash
export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
export DOCKER_HOST="unix://${XDG_RUNTIME_DIR}/podman/podman.sock"
```
or using rootful podman:
```bash
export DOCKER_HOST=unix:///run/podman/podman.sock
```

The socket should be enabled per-default on your system if you have installed podman properly.
You can check if the socket exists with

```bash
ls "${DOCKER_HOST#unix://}"
```

If the file does not show up you can start a rootless podman service with

```bash
podman system service --time 0
```

## Rootless Setup

Rootless podman requires some additional setup to run k3d.

### Using cgroup v2

If you are running cgroup v2, you need to delegate some control groups to normal user processes.
You are using cgroup v2 if `/sys/fs/cgroup/cgroup.controllers` exists.

```bash
ls /sys/fs/cgroup/cgroup.controllers
```

If you are using cgroup v2 you can check which control groups your user has access to (systemd):

```bash
cat /sys/fs/cgroup/user.slice/user-$(id -u).slice/user@$(id -u).service/cgroup.controllers
```

If your output does not include `cpu`, `cpuset`, `io`, `memory`, `pids` then
you need to manually delegate control over these from your init system.

In systemd you can do this by adding `Delegate=cpu cpuset io memory pids` to the `user@` service.

`/etc/systemd/system/user@.service.d/delegate.conf`
```systemd
[Service]
Delegate=cpu cpuset io memory pids
```
```bash
systemctl daemon-reload
```

Reference:  
- [Guide](https://rootlesscontaine.rs/getting-started/common/cgroup2/#enabling-cpu-cpuset-and-io-delegation)  
- [cgroup v2 linux kernel docs](https://www.kernel.org/doc/html/latest/admin-guide/cgroup-v2.html)

### Running the kubelet in rootless mode

You have to inform the k3s kubelet that it should run in rootless mode.  
You can pass the kubelet argument `--kubelet-arg=feature-gates=KubeletInUserNamespace=true`

```yaml
options:
  k3s:
    extraArgs:
      - arg: "--kubelet-arg=feature-gates=KubeletInUserNamespace=true"
        nodeFilters:
          - server:*
          - agent:*
```

!!! note "Issues with k3s image before k3d version v5.9"
    On some distros k3d defaults to the `v1.21.7-k3s1` k3s image which is too old to support this argument.  
    To make it work anyway you can override the image:

    ```yaml
    image: docker.io/rancher/k3s:v1.32.5-k3s1
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

## Creating local registries

Because Podman does not have a default "bridge" network, you have to specify a network using the `--default-network` flag when creating a local registry:

```bash
k3d registry create --default-network podman mycluster-registry
```

To use this registry with a cluster, pass the `--registry-use` flag:

```bash
k3d cluster create --registry-use mycluster-registry mycluster
```

This flag does not have a k3d config option yet.

!!! note "Incompatibility with `--registry-create`"
    Because `--registry-create` assumes the default network to be "bridge", avoid `--registry-create` when using Podman. Instead, always create a registry before creating a cluster.

!!! note "Missing cpuset cgroup controller"
    If you experience an error regarding missing cpuset cgroup controller, ensure the user unit `xdg-document-portal.service` is disabled by running `systemctl --user stop xdg-document-portal.service`. See [this issue](https://github.com/systemd/systemd/issues/18293#issuecomment-831397578)
