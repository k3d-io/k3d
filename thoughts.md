# Thoughts

## commands

### `create`

```shell
k3d
  |- create
  |       |- cluster [NAME ] [flags]
  |       |- node [NAME ] [flags]
  |
  |- delete
  |       |- cluster [NAME ] [flags]
  |       |- node [NAME ] [flags]
  |- get
  |    |- cluster
  |    |- node
  |- start
  |      |- cluster
  |      |- node
  |- stop
  |     |- cluster
  |     |- node
```

## Overview

- `cmd/`: everything around the CLI of k3 = human interface, printed output (e.g. list of clusters)
- `pkg/`: everything else, can be used as a module from other Go projects
  - `cluster/`: everything around managing cluster components
  - `runtimes/`: translate k3d types (node, cluster, etc.) to container runtime specific types and manage them
  - `types/`: collection of types (structs) and constants used by k3d
  - `util/`: utilities, that could be used for everything, not directly related to the project

## k3d <-> runtime

k3d _should_ work with more than one runtime, if we can implement the Runtime interface for it.
Here's how k3d types should translate to a runtime type:

- `cluster` = set of _containers_ running in the same _network_, maybe mounting the same _volume(s)_
- `node` = _container_ with _exposed ports_ and _volume mounts_

### docker

#### node -> container

`container = "github.com/docker/docker/api/types/container"`
`network = "github.com/docker/docker/api/types/network"`

- Name    -> container.Hostname = node.Name
- Role    -> container.Labels["role"] = node.Role
- Image   -> container.Image = node.Image
- Volumes -> container.HostConfig.PortBindings
- Env     -> 
- Args    -> 
- Ports   -> 
- Restart -> 
- Labels  -> container.Labels