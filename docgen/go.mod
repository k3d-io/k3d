module github.com/rancher/k3d/docgen

go 1.16

require (
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/rancher/k3d/v5 v5.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.2.1
	golang.org/x/term v0.0.0-20210406210042-72f3dc4e9b72 // indirect
)

replace github.com/rancher/k3d/v5 => /PATH/TO/YOUR/REPO/DIRECTORY
