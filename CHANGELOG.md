# Changelog

Here, we're tracking changes that belong to specific releases/tags

## v2.0.0

* [DEPRECATION] `--version` flag for `k3d create` was removed in favor of `--image`/`-i` flag (Format: `--image [<repo>/]<image>:<tag>`), where you can include the image version/tag
* [CHANGE] specifying the ApiPort is now only possible via the new `--api-port`/`-a` flag (`--port`/`-p` is being re-used for different functionality, see below)
* [CHANGE] `--port`/`-p` is now an alias for `--publish`, used for mapping arbitrary ports from the host to the cluster containers
    * [DEPRECATION] `--add-port` got removed as an alias for `--publish`
* 