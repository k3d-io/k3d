# Creating multi-server clusters

!!! info "Important note"
    For the best results (and less unexpected issues), choose 1, 3, 5, ... server nodes. (Read more on etcd quorum on [etcd.io](https://etcd.io/docs/v3.3/faq/#why-an-odd-number-of-cluster-members))
    At least 2 cores and 4GiB of RAM are recommended.

## Embedded etcd

Create a cluster with 3 server nodes using k3s' embedded etcd database.
The first server to be created will use the `--cluster-init` flag and k3d will wait for it to be up and running before creating (and connecting) the other server nodes.

```bash
k3d cluster create multiserver --servers 3
```

!!! info "Restarting cluster may fail"
    When you restart the cluster, each node's IP (meaning the underlying container's IP) could change. In this 
    situation, a node might fail to join the existing cluster and consequently fail to start. To address this, 
    you can use the experimental IPAM (IP Address Management) feature to assign each container a static IP. 
    To enable this, create the cluster with the `--subnet auto` or `--subnet 172.45.0.0/16` 
    (or whatever subnet you need) flags. With `--subnet auto`, k3d will create a fake docker network 
    to get an available subnet.
    
    See the relavent issue [#550](https://github.com/k3d-io/k3d/issues/550) for more details.

## Adding server nodes to a running cluster

In theory (and also in practice in most cases), this is as easy as executing the following command:

```bash
k3d node create newserver --cluster multiserver --role server
```

!!! important "There's a trap!"
    If your cluster was initially created with only a single server node, then this will fail.  
    That's because the initial server node was not started with the `--cluster-init` flag and thus is not using the etcd backend.
