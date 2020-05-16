# Creating multi-master clusters

!!! info "Important note"
    For the best results (and less unexpected issues), choose 1, 3, 5, ... master nodes.

## Embedded dqlite

Create a cluster with 3 master nodes using k3s' embedded dqlite database.
The first master to be created will use the `--cluster-init` flag and k3d will wait for it to be up and running before creating (and connecting) the other master nodes.

```bash
    k3d create cluster multimaster --masters 3
```

## Adding master nodes to a running cluster

In theory (and also in practice in most cases), this is as easy as executing the following command:

```bash
    k3d create node newmaster --cluster multimaster --role master
```

!!! important "There's a trap!"
    If your cluster was initially created with only a single master node, then this will fail.
    That's because the initial master node was not started with the `--cluster-init` flag and thus is not using the dqlite backend.
