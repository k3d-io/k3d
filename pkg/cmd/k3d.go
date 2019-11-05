package cmd

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/rancher/k3d/pkg/cmd/checktools"
	"github.com/rancher/k3d/pkg/cmd/create"
	"github.com/rancher/k3d/pkg/cmd/delete"
	"github.com/rancher/k3d/pkg/cmd/getkubeconfig"
	"github.com/rancher/k3d/pkg/cmd/importimages"
	"github.com/rancher/k3d/pkg/cmd/list"
	"github.com/rancher/k3d/pkg/cmd/shell"
	"github.com/rancher/k3d/pkg/cmd/start"
	"github.com/rancher/k3d/pkg/cmd/stop"
	"github.com/rancher/k3d/pkg/cmd/version"
)

type globalOpts struct {
	verbose   bool
	timestamp bool
}

func newCommand() *cobra.Command {
	opts := globalOpts{}
	// rootCmd represents the base command when called without any subcommands
	var rootCmd = &cobra.Command{
		Use:              "k3d",
		Args:             cobra.NoArgs,
		Short:            "Run k3s in Docker!",
		TraverseChildren: true,
		SilenceUsage:     true,
		SilenceErrors:    true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if opts.verbose {
				log.SetLevel(log.DebugLevel)
			} else {
				log.SetLevel(log.InfoLevel)
			}

			if opts.timestamp {
				log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
			}

			return nil
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&opts.timestamp, "timestamp", "", false, "Enable timestmaps in log messages")

	rootCmd.AddCommand(
		checktools.NewCommand(),
		create.NewCommand(),
		delete.NewCommand(),
		getkubeconfig.NewCommand(),
		importimages.NewCommand(),
		list.NewCommand(),
		shell.NewCommand(),
		start.NewCommand(),
		stop.NewCommand(),
		version.NewCommand(),
	)

	return rootCmd
}

// Main executes the root command and returns any errors
func Main() {
	if err := newCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
