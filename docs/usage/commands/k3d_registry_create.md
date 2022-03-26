## k3d registry create

Create a new registry

### Synopsis

Create a new registry.

```
k3d registry create NAME [flags]
```

### Options

```
      --default-network string   Specify the network connected to the registry (default "bridge")
  -h, --help                     help for create
  -i, --image string             Specify image used for the registry (default "docker.io/library/registry:2")
      --no-help                  Disable the help text (How-To use the registry)
  -p, --port [HOST:]HOSTPORT     Select which port the registry should be listening on on your machine (localhost) (Format: [HOST:]HOSTPORT)
                                  - Example: `k3d registry create --port 0.0.0.0:5111` (default "random")
```

### Options inherited from parent commands

```
      --timestamps   Enable Log timestamps
      --trace        Enable super verbose output (trace logging)
      --verbose      Enable verbose output (debug logging)
```

### SEE ALSO

* [k3d registry](k3d_registry.md)	 - Manage registry/registries

