# docgen

Only used to generate the command tree for <https://k3d.io/usage/commands>.

The code will output files in [`../docs/usage/commands/`](../docs/usage/commands/)

## Run

- may required a `replace github.com/rancher/k3d/v4 => PATH/TO/LOCAL/REPO` in the `go.mod`

```bash
# ensure that you're in the docgen dir, as the relative path to the docs/ dir is hardcoded
cd docgen

# run
go run ./main.go
```
