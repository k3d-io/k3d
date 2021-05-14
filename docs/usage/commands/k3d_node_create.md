## k3d node create

Create a new k3s node in docker

### Synopsis

Create a new containerized k3s node (k3s in docker).

```
k3d node create NAME [flags]
```

### Options

```
  -c, --cluster string           Select the cluster that the node shall connect to. (default "k3s-default")
  -h, --help                     help for create
  -i, --image string             Specify k3s image used for the node(s) (default "docker.io/rancher/k3s:v1.20.0-k3s2")
      --k3s-node-label strings   Specify k3s node labels in format "foo=bar"
      --memory string            Memory limit imposed on the node [From docker]
      --replicas int             Number of replicas of this node specification. (default 1)
      --role string              Specify node role [server, agent] (default "agent")
      --timeout duration         Maximum waiting time for '--wait' before canceling/returning.
      --wait                     Wait for the node(s) to be ready before returning.
```

### Options inherited from parent commands

```
      --timestamps   Enable Log timestamps
      --trace        Enable super verbose output (trace logging)
      --verbose      Enable verbose output (debug logging)
```

### SEE ALSO

* [k3d node](k3d_node.md)	 - Manage node(s)

