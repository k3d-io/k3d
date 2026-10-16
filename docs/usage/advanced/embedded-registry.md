# Embedded Registry Mirror (Spegel)

K3s ships an [embedded distributed registry mirror](https://docs.k3s.io/installation/registry-mirror) based on [Spegel](https://github.com/spegel-org/spegel).
With it enabled, nodes share the images they already have with each other, so an image pulled once in the cluster does not have to be pulled from the upstream registry again by every other node.

k3d does not need any special support for this: it only takes a K3s server flag and a `registries.yaml` mirrors entry, both of which k3d can pass through.

## Two things are required

1. The `--embedded-registry` flag on the server node(s).
2. A `mirrors` entry in `registries.yaml` for every registry that should be mirrored.

The flag on its own does nothing. Without a mirrors entry, the server logs this and the mirror never starts:

```text
level=error msg="Not starting distributed registry mirror: no registries configured for distributed mirroring"
```

The [K3s docs](https://docs.k3s.io/installation/registry-mirror#enabling-registry-mirroring) say the same: a node with no registries enabled for mirroring does not take part in the distributed registry at all.

## Using the CLI

Create a `registries.yaml` that lists the registries to mirror. An entry with no endpoints is enough:

```yaml
mirrors:
  docker.io:
```

Then pass the flag to the server and the file to k3d:

```bash
k3d cluster create spegel --agents 2 \
  --k3s-arg "--embedded-registry@server:0" \
  --registry-config registries.yaml
```

The agents do not need the flag; they pick up the embedded registry from the server.

## Using a config file

The same setup in a [SimpleConfig](../configfile.md) file. Here the `"*"` wildcard enables mirroring for all registries:

```yaml
apiVersion: k3d.io/v1alpha5
kind: Simple
metadata:
  name: spegel
servers: 1
agents: 2
options:
  k3s:
    extraArgs:
      - arg: --embedded-registry
        nodeFilters:
          - server:*
registries:
  config: |
    mirrors:
      "*":
```

```bash
k3d cluster create --config spegel-config.yaml
```

## Checking that it works

All k3d nodes share one Docker network, so the ports Spegel uses between nodes (5001 for peer-to-peer, 6443 for the registry itself) are reachable without any port mapping.

Each node's containerd config should now list the local mirror at `127.0.0.1:6443`:

```bash
docker exec k3d-spegel-agent-0 cat /var/lib/rancher/k3s/agent/etc/containerd/certs.d/docker.io/hosts.toml
```

To see an image being served by another node, pull it on one agent first and then on the other, and compare the Spegel cache-hit counter on the second agent before and after:

```bash
kubectl run first --image=docker.io/library/nginx:1.27-alpine --restart=Never \
  --overrides='{"spec":{"nodeName":"k3d-spegel-agent-0"}}'
kubectl wait --for=condition=Ready pod/first

kubectl get --raw /api/v1/nodes/k3d-spegel-agent-1/proxy/metrics | grep '^spegel_mirror_requests_total'

kubectl run second --image=docker.io/library/nginx:1.27-alpine --restart=Never \
  --overrides='{"spec":{"nodeName":"k3d-spegel-agent-1"}}'
kubectl wait --for=condition=Ready pod/second

kubectl get --raw /api/v1/nodes/k3d-spegel-agent-1/proxy/metrics | grep '^spegel_mirror_requests_total'
```

The `cache="hit"` count on `agent-1` goes up when the second pod starts, because the image layers came from `agent-0`.
The `spegel_advertised_image_tags` and `spegel_advertised_content_digests` metrics show what a node is offering to its peers.

!!! info "Tested with"
    k3d built from `main` (v5.9.0-6-g46f3480) and K3s `v1.35.5-k3s1`, one server and two agents.
    Metric names come from the Spegel version bundled with K3s and may differ in other K3s releases.
