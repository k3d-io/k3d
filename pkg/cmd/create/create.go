package create

import (
	"github.com/spf13/cobra"

	run "github.com/rancher/k3d/cli"
	"github.com/rancher/k3d/pkg/constants"
)

type createOptions struct {
	name           string
	volume         []string
	publish        []string
	portAutoOffset int
	version        string
	apiPort        string
	wait           int
	image          string
	serverArg      []string
	agentArg       []string
	env            []string
	workers        int
	autoRestart    bool
}

func NewCommand() *cobra.Command {
	opts := &createOptions{}
	// createCmd represents the create command
	var createCmd = &cobra.Command{
		Use:     "create",
		Args:    cobra.NoArgs,
		Short:   "Create a single- or multi-node k3s cluster in docker containers",
		Aliases: []string{"c"},
		RunE: func(_ *cobra.Command, _ []string) error {
			return run.CreateCluster(
				opts.name,
				opts.image,
				opts.env,
				opts.apiPort,
				opts.serverArg,
				opts.agentArg,
				opts.workers,
				opts.publish,
				opts.volume,
				opts.autoRestart,
				opts.portAutoOffset,
				opts.wait,
			)
		},
	}

	createCmd.Flags().StringVarP(&opts.name, "name", "n", constants.DefaultK3sClusterName, "Set a name for the cluster")
	createCmd.Flags().StringSliceVarP(&opts.volume, "volume", "v", []string{}, "Set a name for the cluster")
	createCmd.Flags().StringSliceVar(&opts.publish, "publish", []string{}, "Publish k3s node ports to the host (Format: `[ip:][host-port:]container-port[/protocol]@node-specifier`, use multiple options to expose more ports)")
	createCmd.Flags().IntVar(&opts.portAutoOffset, "port-auto-offset", 0, "Automatically add an offset (* worker number) to the chosen host port when using `--publish` to map the same container-port from multiple k3d workers to the host")
	createCmd.Flags().StringVar(&opts.version, "version", "", "Choose the k3s image version")
	createCmd.Flags().StringVarP(&opts.apiPort, "api-port", "a", "6443", "Specify the Kubernetes cluster API server port (Format: `[host:]port` (Note: --port/-p will be used for arbitrary port mapping as of v2.0.0, use --api-port/-a instead for setting the api port)")
	createCmd.Flags().IntVarP(&opts.wait, "wait", "t", 0, "Wait for the cluster to come up before returning until timeout (in seconds). Use --wait 0 to wait forever")
	createCmd.Flags().StringVarP(&opts.image, "image", "i", constants.DefaultK3sImage, "Specify a k3s image (Format: <repo>/<image>:<tag>)")

	createCmd.Flags().StringSliceVarP(&opts.serverArg, "server-arg", "x", []string{}, "Pass an additional argument to k3s server (new flag per argument)")
	createCmd.Flags().StringSliceVar(&opts.agentArg, "agent-arg", []string{}, "Pass an additional argument to k3s agent (new flag per argument)")
	createCmd.Flags().StringSliceVarP(&opts.env, "env", "e", []string{}, "Pass an additional environment variable (new flag per variable)")
	createCmd.Flags().IntVarP(&opts.workers, "workers", "w", 0, "Specify how many worker nodes you want to spawn")
	createCmd.Flags().BoolVar(&opts.autoRestart, "auto-restart", false, "Set docker's --restart=unless-stopped flag on the containers")

	return createCmd
}
