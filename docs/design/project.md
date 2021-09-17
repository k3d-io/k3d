# Project Overview

## About This Page

On this page we'll try to give an overview of all the moving bits and pieces in k3d to ease contributions to the project.

## Directory Overview

- [`.github/`](https://github.com/rancher/k3d/tree/main/.github)
  - templates for issues and pull requests
  - GitHub Action workflow definitions
- [`cmd/`](https://github.com/rancher/k3d/tree/main/cmd)
  - everything related to the actual k3d CLI, like the whole command tree, config initialization, argument parsing, etc.
- [`docgen/`](https://github.com/rancher/k3d/tree/main/docgen)
  - sub-module used to auto-generate the documentation for the CLI commands, which ends up in [`docs/usage/commands/`](https://github.com/rancher/k3d/tree/main/docs/usage/commands)
- [`docs/`](https://github.com/rancher/k3d/tree/main/docs)
  - all the resources used to build [k3d.io](https://k3d.io) using mkdocs
- [`pkg/`](<https://github.com/rancher/k3d/tree/main/pkg>)
  - the place where the magic happens.. here you find all the main logic of k3d
  - all function calls within [`cmd/`](https://github.com/rancher/k3d/tree/main/cmd) that do non-trivial things are imported from here
  - this (or rather sub-packages) is what other projects would import as a module to work with k3d without using the CLI
- [`proxy/`](https://github.com/rancher/k3d/tree/main/proxy)
  - configuration to build the [`rancher/k3d-proxy`](https://hub.docker.com/r/rancher/k3d-proxy/) container image which is used as a loadbalancer/proxy in front of (almost) every k3d cluster
  - this is basically just a combination of NGINX with confd and some k3d-specific configuration details
- [`tests/`](https://github.com/rancher/k3d/tree/main/tests)
  - a set of bash scripts used for end-to-end (E2E) tests of k3d
  - mostly used for all the functionality of the k3d CLI which cannot be tested using Go unit tests
- [`tools/`](https://github.com/rancher/k3d/tree/main/tools)
  - sub-module used to build the [`rancher/k3d-tools`](https://hub.docker.com/r/rancher/k3d-tools) container image which supports some k3d functionality like `k3d image import`
- [`vendor/`](https://github.com/rancher/k3d/tree/main/vendor)
  - result of `go mod vendor`, which contains all dependencies of k3d
- [`version/`](https://github.com/rancher/k3d/tree/main/version)
  - package used to code k3d/k3s versions into releases
  - this is where `go build` injects the version tags when building k3d
    - that's the output you see when issuing `k3d version`

## Packages Overview

- [`pkg/`](https://github.com/rancher/k3d/tree/main/pkg)
  - [`actions/`](https://github.com/rancher/k3d/tree/main/pkg/actions)
    - hook actions describing actions (commands, etc.) that run at specific stages of the node/cluster lifecycle
      - e.g. writing configuration files to the container filesystem just before the node (container) starts
  - [`client/`](https://github.com/rancher/k3d/tree/main/pkg/client)
    - all the top level functionality to work with k3d primitives
      - create/retrieve/update/delete/start/stop clusters, nodes, registries, etc. managed by k3d
  - [`config/`](https://github.com/rancher/k3d/tree/main/pkg/config)
    - everything related to the k3d configuration (files), like `SimpleConfig` and `ClusterConfig`
  - [`runtimes/`](https://github.com/rancher/k3d/tree/main/pkg/runtimes)
    - interface and implementations of runtimes that power k3d (currently, that's only Docker)
    - functions in [`client/`](https://github.com/rancher/k3d/tree/main/pkg/client) eventually call runtime functions to "materialize" nodes and clusters
  - [`tools/`](https://github.com/rancher/k3d/tree/main/pkg/tools)
    - functions eventually calling the [`k3d-tools`](https://hub.docker.com/r/rancher/k3d-tools) container (see [`tools/`](https://github.com/rancher/k3d/tree/main/tools) in the repo root)
  - [`types/`](https://github.com/rancher/k3d/tree/main/pkg/types)
    - definition of all k3d primitives and many other details and defaults
    - e.g. contains the definition of a `Node` or a `Cluster` in k3d
  - [`util/`](https://github.com/rancher/k3d/tree/main/pkg/util)
    - some helper functions e.g. for string manipulation/generation, regexp or other re-usable usages

## Anatomy of a Cluster

By default, every k3d cluster consists of at least 2 containers (nodes):

1. (optional, but default and strongly recommended) loadbalancer

   - image: [`rancher/k3d-proxy`](https://hub.docker.com/r/rancher/k3d-proxy/), built from [`proxy/`](https://github.com/rancher/k3d/tree/main/proxy)
   - purpose: proxy and load balance requests from the outside (i.e. most of the times your local host) to the cluster
     - by default, it e.g. proxies all the traffic for the Kubernetes API to port `6443` (default listening port of K3s) to all the server nodes in the cluster
     - can be used for multiple port-mappings to one or more nodes in your cluster
       - that way, port-mappings can also easily be added/removed after the cluster creation, as we can simply re-create the proxy without affecting cluster state

2. (required, always present) primary server node

   - image: [`rancher/k3s`](https://hub.docker.com/r/rancher/k3s/), built from [`github.com/k3s-io/k3s`](https://github.com/k3s-io/k3s)
   - purpose: (initializing) server (formerly: master) node of the cluster
     - runs the K3s executable (which runs containerd, the Kubernetes API Server, etcd/sqlite, etc.): `k3s server`
     - in a multi-server setup, it initializes the cluster with an embedded etcd database (using the K3s `--cluster-init` flag)

3. (optional) secondary server node(s)

   - image: [`rancher/k3s`](https://hub.docker.com/r/rancher/k3s/), built from [`github.com/k3s-io/k3s`](https://github.com/k3s-io/k3s)

4. (optional) agent node(s)

   - image: [`rancher/k3s`](https://hub.docker.com/r/rancher/k3s/), built from [`github.com/k3s-io/k3s`](https://github.com/k3s-io/k3s)
   - purpose: running the K3s agent process (kubelet, etc.): `k3s agent`

## Automation (CI)

The k3d repository mainly leverages the following two CI systems:

- GitHub Actions
  - 2 workflows in <https://github.com/rancher/k3d/tree/main/.github/workflows> to push the artifact to AUR (Arch Linux User Repository)
  - logs/history can be seen in the Actions tab: <https://github.com/rancher/k3d/actions>
- DroneCI
  - a set of pipelines in a single file: <https://github.com/rancher/k3d/blob/main/.drone.yml>
  - static code analysis
  - build
  - tests
  - docker builds + pushes
  - render + push docs
  - (pre-) release to GitHub
  - `push` events end up here (also does the releases, when a tag is pushed): <https://drone-publish.rancher.io/rancher/k3d>
  - `pr`s end up here: <https://drone-pr.rancher.io/rancher/k3d>

## Documentation

The website [k3d.io](https://k3d.io) containing all the documentation for k3d is built using [`mkdocs`](https://www.mkdocs.org/), configured via the [`mkdocs.yml`](https://github.com/rancher/k3d/blob/main/mkdocs.yml) config file with all the content residing in the [`docs/`](https://github.com/rancher/k3d/tree/main/docs) directory (Markdown).  
Use `mkdocs serve` in the repository root to build and serve the webpage locally.  
Some parts of the documentation are being auto-generated, like [`docs/usage/commands/`](https://github.com/rancher/k3d/tree/main/docs/usage/commands) is auto-generated using Cobra's command docs generation functionality in [`docgen/`](https://github.com/rancher/k3d/tree/main/docgen).
