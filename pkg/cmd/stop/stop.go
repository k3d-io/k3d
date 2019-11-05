package stop

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
	// stopCmd represents the stop command
	var stopCmd = &cobra.Command{
		Use:   "stop",
		Args:  cobra.NoArgs,
		Short: "Stop cluster",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run.StopCluster(opts.all, opts.name)
		},
	}

	stopCmd.Flags().StringVarP(&opts.name, "name", "n", constants.DefaultK3sClusterName, "Stop a named cluster")
	stopCmd.Flags().BoolVarP(&opts.all, "all", "a", false, "Stop all running clusters (this ignores the --name/-n flag)")

	return stopCmd
}
