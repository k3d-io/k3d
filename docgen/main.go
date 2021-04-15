package main

import (
	"log"

	"github.com/rancher/k3d/v4/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	k3d := cmd.GetRootCmd()
	k3d.DisableAutoGenTag = true

	if err := doc.GenMarkdownTree(k3d, "../docs/usage/commands"); err != nil {
		log.Fatalln(err)
	}
}
