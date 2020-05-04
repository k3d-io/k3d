# rancher/k3d

k3d is a lightweight wrapper to run [k3s](https://github.com/rancher/k3s) (Rancher Lab's minimal Kubernetes distribution) in docker.

k3d makes it very easy to create single- and multi-node [k3s](https://github.com/rancher/k3s) clusters in docker, e.g. for local development on Kubernetes.

## Requirements

- [docker](https://docs.docker.com/install/)

## Releases

[![Releases](https://img.shields.io/github/release/rancher/k3d.svg)](https://github.com/rancher/k3d/releases/latest)

[![Homebrew](https://img.shields.io/homebrew/v/k3d)](https://formulae.brew.sh/formula/k3d)

## Installation

You have several options there:

- use the install script to grab the latest release:
  - wget: `wget -q -O - https://raw.githubusercontent.com/rancher/k3d/master/install.sh | bash`
  - curl: `curl -s https://raw.githubusercontent.com/rancher/k3d/master/install.sh | bash`
- use the install script to grab a specific release (via `TAG` environment variable):
  - wget: `wget -q -O - https://raw.githubusercontent.com/rancher/k3d/master/install.sh | TAG=v3.0.0-beta.0 bash`
  - curl: `curl -s https://raw.githubusercontent.com/rancher/k3d/master/install.sh | TAG=v3.0.0-beta.0 bash`

- use [Homebrew](https://brew.sh): `brew install k3d` (Homebrew is available for MacOS and Linux)
  - Formula can be found in [homebrew/homebrew-core](https://github.com/Homebrew/homebrew-core/blob/master/Formula/k3d.rb) and is mirrored to [homebrew/linuxbrew-core](https://github.com/Homebrew/linuxbrew-core/blob/master/Formula/k3d.rb)
- install via [AUR](https://aur.archlinux.org/) package [rancher-k3d-bin](https://aur.archlinux.org/packages/rancher-k3d-bin/): `yay -S rancher-k3d-bin`
- grab a release from the [release tab](https://github.com/rancher/k3d/releases) and install it yourself.
- install via go: `go install github.com/rancher/k3d` (**Note**: this will give you unreleased/bleeding-edge changes)

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
