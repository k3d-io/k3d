package getkubeconfig

import (
	run "github.com/rancher/k3d/cli"

	"github.com/spf13/cobra"

	"github.com/rancher/k3d/pkg/constants"
)

type options struct {
	name string
	all  bool
}

func NewCommand() *cobra.Command {
	opts := &options{}

	// getKubeconfigCmd represents the getKubeconfig command
	var getKubeconfigCmd = &cobra.Command{
		Use:     "get-kubeconfig",
		Args:    cobra.NoArgs,
		Short:   "A brief description of your command",
		Aliases: []string{"get-kubeconfig"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return run.GetKubeConfig(opts.all, opts.name)
		},
	}

	getKubeconfigCmd.Flags().StringVarP(&opts.name, "name", "n", constants.DefaultK3sClusterName, "Get kubeconfig for a cluster")
	getKubeconfigCmd.Flags().BoolVarP(&opts.all, "all", "a", false, "Get kubeconfig for all clusters (this ignores the --name/-n flag)")

	return getKubeconfigCmd
}
