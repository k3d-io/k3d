## k3d kubeconfig merge

Write/Merge kubeconfig(s) from cluster(s) into new or existing kubeconfig/file.

### Synopsis

Write/Merge kubeconfig(s) from cluster(s) into new or existing kubeconfig/file.

```
k3d kubeconfig merge [CLUSTER [CLUSTER [...]] | --all] [flags]
```

### Options

```
  -a, --all                         Get kubeconfigs from all existing clusters
  -h, --help                        help for merge
  -d, --kubeconfig-merge-default    Merge into the default kubeconfig ($KUBECONFIG or /home/thklein/.kube/config)
  -s, --kubeconfig-switch-context   Switch to new context (default true)
  -o, --output string               Define output [ - | FILE ] (default from $KUBECONFIG or /home/thklein/.kube/config
      --overwrite                   [Careful!] Overwrite existing file, ignoring its contents
  -u, --update                      Update conflicting fields in existing kubeconfig (default true)
```

### Options inherited from parent commands

```
      --timestamps   Enable Log timestamps
      --trace        Enable super verbose output (trace logging)
      --verbose      Enable verbose output (debug logging)
```

### SEE ALSO

* [k3d kubeconfig](k3d_kubeconfig.md)	 - Manage kubeconfig(s)

