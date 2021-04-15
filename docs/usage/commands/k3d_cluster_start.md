## k3d cluster start

Start existing k3d cluster(s)

### Synopsis

Start existing k3d cluster(s)

```
k3d cluster start [NAME [NAME...] | --all] [flags]
```

### Options

```
  -a, --all                Start all existing clusters
  -h, --help               help for start
      --timeout duration   Maximum waiting time for '--wait' before canceling/returning.
      --wait               Wait for the server(s) (and loadbalancer) to be ready before returning. (default true)
```

### Options inherited from parent commands

```
      --timestamps   Enable Log timestamps
      --trace        Enable super verbose output (trace logging)
      --verbose      Enable verbose output (debug logging)
```

### SEE ALSO

* [k3d cluster](k3d_cluster.md)	 - Manage cluster(s)

