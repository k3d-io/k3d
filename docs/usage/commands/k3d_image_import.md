## k3d image import

Import image(s) from docker into k3d cluster(s).

### Synopsis

Import image(s) from docker into k3d cluster(s).

```
k3d image import [IMAGE | ARCHIVE [IMAGE | ARCHIVE...]] [flags]
```

### Options

```
  -c, --cluster stringArray   Select clusters to load the image to. (default [k3s-default])
  -h, --help                  help for import
  -k, --keep-tarball          Do not delete the tarball containing the saved images from the shared volume
```

### Options inherited from parent commands

```
      --timestamps   Enable Log timestamps
      --trace        Enable super verbose output (trace logging)
      --verbose      Enable verbose output (debug logging)
```

### SEE ALSO

* [k3d image](k3d_image.md)	 - Handle container images.

