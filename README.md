# k3d-go

[![Build Status](https://travis-ci.com/iwilltry42/k3d-go.svg?branch=master)](https://travis-ci.com/iwilltry42/k3d-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/iwilltry42/k3d-go)](https://goreportcard.com/report/github.com/iwilltry42/k3d-go)

## k3s in docker

k3s is the lightweight Kubernetes distribution by Rancher: [rancher/k3s](https://github.com/rancher/k3s)

This repository is basically [zeerorg/k3s-in-docker](https://github.com/zeerorg/k3s-in-docker) reimplemented in Golang with some different/new functionality... just because I didn't have time to learn Rust.

Thanks to @zeerorg for the original work!

## Requirements

- docker

## Install

You have several options there:

- use the install script to grab the latest release: 
  - wget: `wget -q -O - https://raw.githubusercontent.com/iwilltry42/k3d-go/master/install.sh | bash`
  - curl: `curl -s https://raw.githubusercontent.com/iwilltry42/k3d-go/master/install.sh | bash`
- Grab a release from the [release tab](https://github.com/iwilltry42/k3d-go/releases) and install it yourself.
- Via go: `go install github.com/iwilltry42/k3d-go`

or...

## Build

1. Clone this repo, e.g. via `go get -u github.com/iwilltry42/k3d-go/releases`
2. Inside the repo run
   - `make` to build for your current system
   - `go install` to install it to your `GOPATH`
   - `make build-cross` to build for all systems

## Usage

Check out what you can do via `k3d help`

Example Workflow: Create a new cluster and use it with `kubectl`

1. `k3d create` to create a new single-node cluster (docker container)
2. `export KUBECONFIG=$(k3d get-kubeconfig)` to make `kubectl` to use the kubeconfig for that cluster
3. execute some commands like `kubectl get pods --all-namespaces`
4. `k3d delete` to delete the default cluster

## TODO

- [ ] Use the docker client library instead of commands
- [ ] Test the docker version
- [ ] Improve cluster state management
- [ ] Use [sirupsen/logrus](https://github.com/sirupsen/logrus) for prettier logs
- [ ] Add install script