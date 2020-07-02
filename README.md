# [![k3d](docs/static/img/k3d_logo_black_blue.svg)](https://k3d.io/)

[![Build Status](https://img.shields.io/drone/build/rancher/k3d/master?logo=drone&server=https%3A%2F%2Fdrone-publish.rancher.io&style=flat-square)](https://drone-publish.rancher.io/rancher/k3d)
[![License](https://img.shields.io/github/license/rancher/k3d?style=flat-square)](./LICENSE.md)
![Downloads](https://img.shields.io/github/downloads/rancher/k3d/total.svg?style=flat-square)

[![Go Module](https://img.shields.io/badge/Go%20Module-github.com%2Francher%2Fk3d%2Fv3-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/rancher/k3d/v3)
[![Go version](https://img.shields.io/github/go-mod/go-version/rancher/k3d?logo=go&logoColor=white&style=flat-square)](./go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/rancher/k3d?style=flat-square)](https://goreportcard.com/report/github.com/rancher/k3d)

**Please Note:** `master` is now v3.0.0 and the code for v1.x can be found in the `master-v1` branch!

## [k3s in docker](https://k3d.io)

k3s is the lightweight Kubernetes distribution by Rancher: [rancher/k3s](https://github.com/rancher/k3s)

k3d creates containerized k3s clusters. This means, that you can spin up a multi-node k3s cluster on a single machine using docker.

[![asciicast](https://asciinema.org/a/330413.svg)](https://asciinema.org/a/330413)

## Learning

- Website with documentation: [k3d.io](https://k3d.io/)
- [Rancher Meetup - May 2020 - Simplifying Your Cloud-Native Development Workflow With K3s, K3c and K3d (YouTube)](https://www.youtube.com/watch?v=hMr3prm9gDM)
  - k3d demo repository: [iwilltry42/k3d-demo](https://github.com/iwilltry42/k3d-demo)

## Requirements

- [docker](https://docs.docker.com/install/)

## Releases

**Note**: In May 2020 we upgraded from v1.7.x to **v3.0.0** after a complete rewrite of k3d!

| Platform | Stage | Version | Release Date |  |
|-----------------|--------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------|---|
| [**GitHub Releases**](https://github.com/rancher/k3d/releases) | stable | [![GitHub release (latest by date)](https://img.shields.io/github/v/release/rancher/k3d?label=%20&style=for-the-badge&logo=github)](https://github.com/rancher/k3d/releases/latest) | [![GitHub Release Date](https://img.shields.io/github/release-date/rancher/k3d?label=%20&style=for-the-badge)](https://github.com/rancher/k3d/releases/latest) |  |
| [**GitHub Releases**](https://github.com/rancher/k3d/releases) | latest | [![GitHub release (latest by date including pre-releases)](https://img.shields.io/github/v/release/rancher/k3d?include_prereleases&label=%20&style=for-the-badge&logo=github)](https://github.com/rancher/k3d/releases) | [![GitHub (Pre-)Release Date](https://img.shields.io/github/release-date-pre/rancher/k3d?label=%20&style=for-the-badge)](https://github.com/rancher/k3d/releases) |  |
| [**Homebrew**](https://formulae.brew.sh/formula/k3d) | - | [![homebrew](https://img.shields.io/homebrew/v/k3d?label=%20&style=for-the-badge)](https://formulae.brew.sh/formula/k3d) | - |  |

## Get

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

or...

## Build

1. Clone this repo, e.g. via `git clone git@github.com:rancher/k3d.git` or `go get github.com/rancher/k3d/v3@master`
2. Inside the repo run
   - 'make install-tools' to make sure required go packages are installed
3. Inside the repo run one of the following commands
   - `make build` to build for your current system
   - `go install` to install it to your `GOPATH` (**Note**: this will give you unreleased/bleeding-edge changes)
   - `make build-cross` to build for all systems

## Usage

Check out what you can do via `k3d help` or check the docs @ [k3d.io](https://k3d.io)

Example Workflow: Create a new cluster and use it with `kubectl`

1. `k3d create cluster CLUSTER_NAME` to create a new single-node cluster (= 1 container running k3s)
2. `k3d get-kubeconfig CLUSTER_NAME --switch` to update your default kubeconfig and switch the current-context to the new one
3. execute some commands like `kubectl get pods --all-namespaces`
4. `k3d delete cluster CLUSTER_NAME` to delete the default cluster

## Connect

1. Join the Rancher community on slack via [slack.rancher.io](https://slack.rancher.io/)
2. Go to [rancher-users.slack.com](https://rancher-users.slack.com) and join our channel #k3d
3. Start chatting

## History

This repository is based on [@zeerorg](https://github.com/zeerorg/)'s [zeerorg/k3s-in-docker](https://github.com/zeerorg/k3s-in-docker), reimplemented in Go by [@iwilltry42](https://github.com/iwilltry42/) in [iwilltry42/k3d](https://github.com/iwilltry42/k3d), which got adopted by Rancher in[rancher/k3d](https://github.com/rancher/k3d).

## Related Projects

* [k3x](https://github.com/inercia/k3x): a graphics interface (for Linux) to k3d.
