# Importing modes

## Auto

Auto-determine whether to use `direct` or `tools-node`.

For remote container runtimes, `tools-node` is faster due to less network overhead, thus it is automatically selected for remote runtimes.

Otherwise direct is used.

## Direct

Directly load the given images to the k3s nodes. No separate container is spawned, no intermediate files are written.

## Tools Node

Start a `k3d-tools` container in the container runtime, copy images to that runtime, then load the images to k3s nodes from there.

