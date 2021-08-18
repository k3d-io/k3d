package main

import (
	"github.com/rancher/k3d/v4/cmd"
	l "github.com/rancher/k3d/v4/pkg/logger"
	"github.com/spf13/cobra/doc"
)

func main() {
	k3d := cmd.NewCmdK3d()
	k3d.DisableAutoGenTag = true

	if err := doc.GenMarkdownTree(k3d, "../docs/usage/commands"); err != nil {
		l.Log().Fatalln(err)
	}
}
