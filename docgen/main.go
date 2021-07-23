package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/rancher/k3d/v4/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	k3d := cmd.NewCmdK3d()
	k3d.DisableAutoGenTag = true

	if err := doc.GenMarkdownTree(k3d, "../docs/usage/commands"); err != nil {
		log.Fatalln(err)
	}
}
