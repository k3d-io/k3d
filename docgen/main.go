package main

import (
	"github.com/k3d-io/k3d/v5/cmd"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/spf13/cobra/doc"
)

func main() {
	k3d := cmd.NewCmdK3d()
	k3d.DisableAutoGenTag = true

	if err := doc.GenMarkdownTree(k3d, "./docs/usage/commands"); err != nil {
		l.Log().Fatalln(err)
	}
}
