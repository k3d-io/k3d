## k3d version list

List k3d/K3s versions

```
k3d version list [flags]
```

### Options

```
  -e, --exclude string   Exclude Regexp (default excludes pre-releases and arch-specific tags) (default ".+(rc|engine|alpha|beta|dev|test|arm|arm64|amd64).*")
  -f, --format string    Output Format (default "raw")
  -h, --help             help for list
  -i, --include string   Include Regexp (default includes everything (default ".*")
  -l, --limit int        Limit number of tags in output (0 = unlimited)
  -s, --sort string      Sort Mode (asc | desc | off) (default "desc")
```

### Options inherited from parent commands

```
      --timestamps   Enable Log timestamps
      --trace        Enable super verbose output (trace logging)
      --verbose      Enable verbose output (debug logging)
```

### SEE ALSO

* [k3d version](k3d_version.md)	 - Show k3d and default k3s version

