package list

import (
	run "github.com/rancher/k3d/cli"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	// listCmd represents the list command
	var listCmd = &cobra.Command{
		Use:     "list",
		Args:    cobra.NoArgs,
		Short:   "List all clusters",
		Aliases: []string{"l"},
		RunE: func(_ *cobra.Command, _ []string) error {
			return run.ListClusters()
		},
	}

	return listCmd

}
