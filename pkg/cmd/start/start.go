package start

import (
	"github.com/spf13/cobra"

	run "github.com/rancher/k3d/cli"
	"github.com/rancher/k3d/pkg/constants"
)

type options struct {
	name string
	all  bool
}

func NewCommand() *cobra.Command {
	opts := &options{}
	// startCmd represents the start command
	var startCmd = &cobra.Command{
		Use:     "start",
		Args:    cobra.NoArgs,
		Short:   "Start a stopped cluster",
		Aliases: []string{"start"},
		RunE: func(_ *cobra.Command, _ []string) error {
			return run.StartCluster(opts.all, opts.name)
		},
	}

	startCmd.Flags().StringVarP(&opts.name, "name", "n", constants.DefaultK3sClusterName, "start a cluster by name")
	startCmd.Flags().BoolVarP(&opts.all, "all", "a", false, "Start all stopped clusters (this ignores the --name/-n flag)")
	return startCmd
}
