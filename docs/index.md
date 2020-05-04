# rancher/k3d

k3d is a lightweight wrapper to run [k3s](https://github.com/rancher/k3s) (Rancher Lab's minimal Kubernetes distribution) in docker.

k3d makes it very easy to create single- and multi-node [k3s](https://github.com/rancher/k3s) clusters in docker, e.g. for local development on Kubernetes.

## Installation

- Install via brew (homebrew/linuxbrew): `brew install k3d`
- Install via script:
  - wget: `wget -q -O - https://raw.githubusercontent.com/rancher/k3d/master/install.sh | bash`
  - curl: `curl -s https://raw.githubusercontent.com/rancher/k3d/master/install.sh | bash`
- Grab a release from the [release tab](https://github.com/rancher/k3d/releases) and install it yourself.
- Get via go: `go install github.com/rancher/k3d` (**Note**: this will give you unreleased/bleeding-edge changes)

## Quick Start

Create a cluster named `mycluster` with just a single master node:

```bash
k3d create cluster mycluster
```

Get the new cluster's connection details merged into your default kubeconfig (usually specified using the `KUBECONFIG` environment variable or the default path `$HOME/.kube/config`) and directly switch to the new context:

```bash
k3d get kubeconfig mycluster --switch
```

Use the new cluster with [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/), e.g.:

```bash
kubectl get nodes
```
