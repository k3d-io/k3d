# Creating multi-server clusters

!!! info "Important note"
    For the best results (and less unexpected issues), choose 1, 3, 5, ... server nodes.
    At least 2 cores and 4GiB of RAM are recommended.

## Embedded etcd (old: dqlite)

Create a cluster with 3 server nodes using k3s' embedded etcd (old: dqlite) database.
The first server to be created will use the `--cluster-init` flag and k3d will wait for it to be up and running before creating (and connecting) the other server nodes.

```bash
k3d cluster create multiserver --servers 3
```

## Adding server nodes to a running cluster

In theory (and also in practice in most cases), this is as easy as executing the following command:

```bash
k3d node create newserver --cluster multiserver --role server
```

!!! important "There's a trap!"
    If your cluster was initially created with only a single server node, then this will fail.  
    That's because the initial server node was not started with the `--cluster-init` flag and thus is not using the etcd (old: dqlite) backend.
