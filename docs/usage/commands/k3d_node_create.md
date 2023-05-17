## k3d node create

Create a new k3s node in docker

### Synopsis

Create a new containerized k3s node (k3s in docker).

```
k3d node create NAME [flags]
```

### Options

```
  -c, --cluster string           Cluster URL or k3d cluster name to connect to. (default "k3s-default")
  -h, --help                     help for create
  -i, --image string             Specify k3s image used for the node(s) (default: copied from existing node)
      --k3s-arg stringArray      Additional args passed to k3d command
      --k3s-node-label strings   Specify k3s node labels in format "foo=bar"
      --memory string            Memory limit imposed on the node [From docker]
  -n, --network strings          Add node to (another) runtime network
      --replicas int             Number of replicas of this node specification. (default 1)
      --role string              Specify node role [server, agent] (default "agent")
      --runtime-label strings    Specify container runtime labels in format "foo=bar"
      --runtime-ulimit strings   Specify container runtime ulimit in format "ulimit=soft:hard"
      --timeout duration         Maximum waiting time for '--wait' before canceling/returning.
  -t, --token string             Override cluster token (required when connecting to an external cluster)
      --wait                     Wait for the node(s) to be ready before returning. (default true)
```

### Options inherited from parent commands

```
      --timestamps   Enable Log timestamps
      --trace        Enable super verbose output (trace logging)
      --verbose      Enable verbose output (debug logging)
```

### SEE ALSO

* [k3d node](k3d_node.md)	 - Manage node(s)

