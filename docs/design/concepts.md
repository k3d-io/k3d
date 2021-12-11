# Concepts

## Nodefilters

### About

Nodefilters are a concept in k3d to specify which nodes of a newly created cluster a condition or setting should apply to.

### Syntax

The overall syntax is `@<group>:<subset>[:<suffix]`.

- `@` denotes the start of a nodefilter in a k3d flag value
- `<group>` denotes the node group you want to filter in
  - one of `server`, `servers`, `agent`, `agents`, `loadbalancer`, `all`
    - note, that `all` also includes the cluster-external server loadbalancer (`k3d-proxy` container)
- `<subset>` denotes the subset of the chosen group you want to apply the flag to
  - wildcard `*`: all nodes in that group
  - index, e.g. `0`: only the first node of that group
  - list, e.g. `1,3,5`: nodes 1, 3 and 5 of that group
  - range, e.g. `2-4`: nodes 2 to 4 of that group
- `<suffix>` (optional) can hold some flag specific configuration
  - e.g. for the `--port` flag this could be `direct` or `proxy` (default) to configure the way of exposing ports

### Example

- Problem: You want to have Nginx as your ingress controller, but by default, K3s deploys Traefik.
- Solution: Disabling the default Traefik deployment using K3s' `--disable=traefik` flag.
- Note: It's enough to do this on the first (initializing) server node.
- How-To: `k3d cluster create notraefik --k3s-arg="--disable=traefik@server:0"`
  - Looking at `--k3s-arg="--disable=traefik@server:0"`, everything after the `@` sign is part of the nodefilter.
    - `server` is the node group: server nodes
    - after the `:` follows the subset, which in this case is the index `0`: the first server node to be created (`k3d-notraefik-server-0`)
