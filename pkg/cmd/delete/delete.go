package delete

import (
	run "github.com/rancher/k3d/cli"

	"github.com/spf13/cobra"

	"github.com/rancher/k3d/pkg/constants"
)

type deleteOptions struct {
	name string
	all  bool
}

func NewCommand() *cobra.Command {
	opts := &deleteOptions{}
	var deleteCmd = &cobra.Command{
		Use:     "delete",
		Args:    cobra.NoArgs,
		Short:   "Delete cluster",
		Aliases: []string{"d", "del"},
		RunE: func(_ *cobra.Command, _ []string) error {
			return run.DeleteCluster(opts.all, opts.name)
		},
	}

	deleteCmd.Flags().StringVarP(&opts.name, "name", "n", constants.DefaultK3sClusterName, "Delete a cluster by name")
	deleteCmd.Flags().BoolVarP(&opts.all, "all", "a", false, "Delete all existing clusters (this ignores the --name/-n flag)")

	return deleteCmd

}
