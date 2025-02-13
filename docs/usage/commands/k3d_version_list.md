## k3d version list

List k3d/K3s versions. Component can be one of 'k3d', 'k3s', 'k3d-proxy', 'k3d-tools'.

```
k3d version list COMPONENT [flags]
```

### Options

```
  -e, --exclude string   Exclude Regexp (default excludes pre-releases and arch-specific tags) (default ".+(rc|engine|alpha|beta|dev|test|arm|arm64|amd64|s390x).*")
  -f, --format string    [DEPRECATED] Use --output instead (default "raw")
  -h, --help             help for list
  -i, --include string   Include Regexp (default includes everything (default ".*")
  -l, --limit int        Limit number of tags in output (0 = unlimited)
  -o, --output string    Output Format [raw | repo] (default "raw")
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

