## k3d image import

Import image(s) from docker into k3d cluster(s).

### Synopsis

Import image(s) from docker into k3d cluster(s).

If an IMAGE starts with the prefix 'docker.io/', then this prefix is stripped internally.
That is, 'docker.io/k3d-io/k3d-tools:latest' is treated as 'k3d-io/k3d-tools:latest'.

If an IMAGE starts with the prefix 'library/' (or 'docker.io/library/'), then this prefix is stripped internally.
That is, 'library/busybox:latest' (or 'docker.io/library/busybox:latest') are treated as 'busybox:latest'.

If an IMAGE does not have a version tag, then ':latest' is assumed.
That is, 'k3d-io/k3d-tools' is treated as 'k3d-io/k3d-tools:latest'.

A file ARCHIVE always takes precedence.
So if a file './k3d-io/k3d-tools' exists, k3d will try to import it instead of the IMAGE of the same name.

```
k3d image import [IMAGE | ARCHIVE [IMAGE | ARCHIVE...]] [flags]
```

### Options

```
  -c, --cluster stringArray   Select clusters to load the image to. (default [k3s-default])
  -h, --help                  help for import
  -k, --keep-tarball          Do not delete the tarball containing the saved images from the shared volume
  -t, --keep-tools            Do not delete the tools node after import
  -m, --mode string           Which method to use to import images into the cluster [auto, direct, tools]. See https://k3d.io/stable/usage/importing_images/ (default "tools-node")
```

### Options inherited from parent commands

```
      --timestamps   Enable Log timestamps
      --trace        Enable super verbose output (trace logging)
      --verbose      Enable verbose output (debug logging)
```

### SEE ALSO

* [k3d image](k3d_image.md)	 - Handle container images.

