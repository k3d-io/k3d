## k3d node edit

[EXPERIMENTAL] Edit node(s).

### Synopsis

[EXPERIMENTAL] Edit node(s).

```
k3d node edit NODE [flags]
```

### Options

```
  -h, --help                                                               help for edit
      --port-add [HOST:][HOSTPORT:]CONTAINERPORT[/PROTOCOL][@NODEFILTER]   [EXPERIMENTAL] (serverlb only!) Map ports from the node container to the host (Format: [HOST:][HOSTPORT:]CONTAINERPORT[/PROTOCOL][@NODEFILTER])
                                                                            - Example: `k3d node edit k3d-mycluster-serverlb --port-add 8080:80`
```

### Options inherited from parent commands

```
      --timestamps   Enable Log timestamps
      --trace        Enable super verbose output (trace logging)
      --verbose      Enable verbose output (debug logging)
```

### SEE ALSO

* [k3d node](k3d_node.md)	 - Manage node(s)

