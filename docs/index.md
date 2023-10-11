# Overview

![k3d](static/img/k3d_logo_black_blue.svg)

## What is k3d?

k3d is a lightweight wrapper to run [k3s](https://github.com/rancher/k3s) (Rancher Lab's minimal Kubernetes distribution) in docker.

k3d makes it very easy to create single- and multi-node [k3s](https://github.com/rancher/k3s) clusters in docker, e.g. for local development on Kubernetes.

**Note:** k3d is a **community-driven project** but it's not an official Rancher (SUSE) product.
**Sponsoring**: To spend any significant amount of time improving k3d, we rely on sponsorships:

  - [**GitHub Sponsors**: ![GitHub Sponsors](https://img.shields.io/github/sponsors/k3d-io?label=GitHub%20Sponsors&style=flat-square)](https://github.com/sponsors/k3d-io)
  - [**LiberaPay**: ![Liberapay patrons](https://img.shields.io/liberapay/patrons/k3d-io?label=Liberapay%20Patrons&style=flat-square)](https://liberapay.com/k3d-io)
  - **IssueHunt**: <https://issuehunt.io/r/k3d-io/k3d>

??? Tip "View a quick demo"
    <asciinema-player src="/static/asciicast/20210917_k3d_v5.0.0_01.cast" cols=200 rows=32></asciinema-player>

## Learning

!!! Tip "k3d demo repository: [iwilltry42/k3d-demo](https://github.com/iwilltry42/k3d-demo)"
    Featured use-cases include:

    - **hot-reloading** of code when developing on k3d (Python Flask App)
    - build-deploy-test cycle using **Tilt**
    - full cluster lifecycle for simple and **multi-server** clusters
    - Proof of Concept of using k3d as a service in **Drone CI**

- [Rancher Meetup - May 2020 - Simplifying Your Cloud-Native Development Workflow With K3s, K3c and K3d (YouTube)](https://www.youtube.com/watch?v=hMr3prm9gDM)

## Requirements

- [**docker**](https://docs.docker.com/install/) to be able to use k3d at all
  - Note: k3d v5.x.x requires at least Docker v20.10.5 (runc >= v1.0.0-rc93) to work properly (see [#807](https://github.com/k3d-io/k3d/issues/807))
- [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) to interact with the Kubernetes cluster

## Releases

| Platform | Stage | Version | Release Date | Downloads so far |
|-----------------|--------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------|---|
| [**GitHub Releases**](https://github.com/k3d-io/k3d/releases) | stable | [![GitHub release (latest by date)](https://img.shields.io/github/v/release/k3d-io/k3d?label=%20&style=for-the-badge&logo=github)](https://github.com/k3d-io/k3d/releases/latest) | [![GitHub Release Date](https://img.shields.io/github/release-date/k3d-io/k3d?label=%20&style=for-the-badge)](https://github.com/k3d-io/k3d/releases/latest) | ![GitHub Release Downloads](https://img.shields.io/github/downloads/k3d-io/k3d/latest/total?label=%20&style=for-the-badge) |
| [**GitHub Releases**](https://github.com/k3d-io/k3d/releases) | latest | [![GitHub release (latest by date including pre-releases)](https://img.shields.io/github/v/release/k3d-io/k3d?include_prereleases&label=%20&style=for-the-badge&logo=github)](https://github.com/k3d-io/k3d/releases) | [![GitHub (Pre-)Release Date](https://img.shields.io/github/release-date-pre/k3d-io/k3d?label=%20&style=for-the-badge)](https://github.com/k3d-io/k3d/releases) | ![GitHub Release Downloads (incl. Pre-Releases)](https://img.shields.io/github/downloads-pre/k3d-io/k3d/latest/total?label=%20&style=for-the-badge) |
| [**Homebrew**](https://formulae.brew.sh/formula/k3d) | stable | [![homebrew](https://img.shields.io/homebrew/v/k3d?label=%20&style=for-the-badge)](https://formulae.brew.sh/formula/k3d) | - | - |
| [**Chocolatey**](https://chocolatey.org/packages/k3d/)| stable | [![chocolatey](https://img.shields.io/chocolatey/v/k3d?label=%20&style=for-the-badge)](https://chocolatey.org/packages/k3d/) | - | - |
| [**Scoop**](https://github.com/ScoopInstaller/Main/blob/master/bucket/k3d.json/)| stable | [![scoop](https://img.shields.io/scoop/v/k3d?label=%20&style=for-the-badge)](https://github.com/ScoopInstaller/Main/blob/master/bucket/k3d.json/) | - | - |

## Installation

You have several options there:

### [:fontawesome-regular-file-code: Install Script](https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh)

#### Install current latest release

- wget:

    ```bash
    wget -q -O - https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
    ```

- curl:

    ```bash
    curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
    ```

#### Install specific release

Use the install script to grab a specific release (via `TAG` environment variable):

- wget:

    ```bash
    wget -q -O - https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | TAG=v5.0.0 bash
    ```

- curl:

    ```bash
    curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | TAG=v5.0.0 bash
    ```

### Other Installers

??? Tip "Other Installation Methods"

    - [:fontawesome-solid-beer: Homebrew (MacOS/Linux)](https://brew.sh):
        ```bash
        brew install k3d
        ```

        *Note*: The formula can be found in [homebrew/homebrew-core](https://github.com/Homebrew/homebrew-core/blob/master/Formula/k3d.rb) and is mirrored to [homebrew/linuxbrew-core](https://github.com/Homebrew/linuxbrew-core/blob/master/Formula/k3d.rb)

    - [:material-arch: AUR (Arch Linux User Repository)](https://aur.archlinux.org/):
        ```bash
        yay -S rancher-k3d-bin
        ```

        Package [rancher-k3d-bin](https://aur.archlinux.org/packages/rancher-k3d-bin/)

    - [:material-github: Download GitHub Release](https://github.com/k3d-io/k3d/releases)

      Grab a release binary from the [release tab](https://github.com/k3d-io/k3d/releases) and install it yourself

    - [:material-microsoft-windows: Chocolatey (Windows)](https://chocolatey.org/): `choco install k3d`

      *Note*: package source can be found in [erwinkersten/chocolatey-packages](https://github.com/erwinkersten/chocolatey-packages/tree/master/automatic/k3d)

    - [:material-microsoft-windows: Scoop (Windows)](https://scoop.sh//): `scoop install k3d`

      *Note*: package source can be found in [ScoopInstaller/Main](https://github.com/ScoopInstaller/Main/blob/master/bucket/k3d.json)

    - [arkade](https://github.com/alexellis/arkade): `arkade get k3d`

    - [asdf](https://asdf-vm.com): `asdf plugin-add k3d && asdf install k3d latest`

      *Note*: `asdf plugin-add k3d`, then `asdf install k3d <tag>` with `<tag> = latest` or `5.x.x` for a specific version (maintained by [spencergilbert/asdf-k3d](https://github.com/spencergilbert/asdf-k3d))

    - Others
      - install via go: `#!bash go install github.com/k3d-io/k3d/v5@latest` (**Note**: this will give you unreleased/bleeding-edge changes)

## Quick Start

Create a cluster named `mycluster` with just a single server node:

```bash
k3d cluster create mycluster
```

Use the new cluster with [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/), e.g.:

```bash
kubectl get nodes
```

??? Note "Getting the cluster's kubeconfig (included in `k3d cluster create`)"
    Get the new cluster's connection details merged into your default kubeconfig (usually specified using the `KUBECONFIG` environment variable or the default path `#!bash $HOME/.kube/config`) and directly switch to the new context:

    ```bash
    k3d kubeconfig merge mycluster --kubeconfig-switch-context
    ```

## Connect

1. Join the Rancher community on slack via [slack.rancher.io](https://slack.rancher.io/)
2. Go to [rancher-users.slack.com](https://rancher-users.slack.com) and join our channel #k3d
3. Start chatting

## Related Projects

- [vscode-k3d](https://github.com/inercia/vscode-k3d/): VSCode Extension to handle k3d clusters from within VSCode
- [k3x](https://github.com/inercia/k3x): a graphics interface (for Linux) to k3d.
- [AbsaOSS/k3d-action](https://github.com/AbsaOSS/k3d-action): fully customizable GitHub Action to run lightweight Kubernetes clusters.
- [AutoK3s](https://github.com/cnrancher/autok3s): a lightweight tool to help run K3s everywhere including k3d provider.
- [nolar/setup-k3d-k3s](https://github.com/nolar/setup-k3d-k3s): setup K3d/K3s for GitHub Actions.
