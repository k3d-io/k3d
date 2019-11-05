package checktools

import (
	run "github.com/rancher/k3d/cli"

	"github.com/spf13/cobra"
)

// NewCommand returns the checkTools command
func NewCommand() *cobra.Command {
	var checkToolsCmd = &cobra.Command{
		Use:     "check-tools",
		Args:    cobra.NoArgs,
		Short:   "A brief description of your command",
		Aliases: []string{"check-tools", "ct"},
		RunE: func(_ *cobra.Command, _ []string) error {
			return run.CheckTools()
		},
	}

	return checkToolsCmd
}
