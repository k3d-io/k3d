package shell

import (
	"strings"

	run "github.com/rancher/k3d/cli"
	"github.com/spf13/cobra"

	"github.com/rancher/k3d/pkg/constants"
)

type options struct {
	name  string
	shell string
}

func NewCommand() *cobra.Command {
	opts := &options{}
	// shellCmd represents the shell command
	var shellCmd = &cobra.Command{
		Use:   "shell",
		Short: "Start a subshell for a cluster",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			return run.Shell(opts.name, opts.shell, strings.Join(args, " "))
		},
	}

	shellCmd.Flags().StringVarP(&opts.name, "name", "n", constants.DefaultK3sClusterName, "Start a subshell for a cluster")
	shellCmd.Flags().StringVarP(&opts.shell, "shell", "s", "auto", "which shell to use. One of [auto, bash, zsh]")

	return shellCmd

}
