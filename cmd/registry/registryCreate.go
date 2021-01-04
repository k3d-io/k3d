/*
Copyright © 2020 The k3d Author(s)

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package registry

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/rancher/k3d/v4/pkg/runtimes"
	k3d "github.com/rancher/k3d/v4/pkg/types"

	"github.com/rancher/k3d/v4/pkg/client"

	cliutil "github.com/rancher/k3d/v4/cmd/util"
	"github.com/spf13/cobra"
)

type regCreatePreProcessedFlags struct {
	Port     string
	Clusters []string
}

type regCreateFlags struct {
	Image string
}

// NewCmdRegistryCreate returns a new cobra command
func NewCmdRegistryCreate() *cobra.Command {

	flags := &regCreateFlags{}
	ppFlags := &regCreatePreProcessedFlags{}

	// create new command
	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a new registry",
		Long:  `Create a new registry.`,
		Args:  cobra.MaximumNArgs(1), // maximum one name accepted
		Run: func(cmd *cobra.Command, args []string) {
			reg, clusters := parseCreateRegistryCmd(cmd, args, flags, ppFlags)
			regNode, err := client.RegistryRun(cmd.Context(), runtimes.SelectedRuntime, reg)
			if err != nil {
				log.Fatalln(err)
			}
			if err := client.RegistryConnectClusters(cmd.Context(), runtimes.SelectedRuntime, regNode, clusters); err != nil {
				log.Errorln(err)
			}
		},
	}

	// add flags

	// TODO: connecting to clusters requires non-existing config reload functionality in containerd
	cmd.Flags().StringArrayVarP(&ppFlags.Clusters, "cluster", "c", nil, "[NotReady] Select the cluster(s) that the registry shall connect to.")
	if err := cmd.RegisterFlagCompletionFunc("cluster", cliutil.ValidArgsAvailableClusters); err != nil {
		log.Fatalln("Failed to register flag completion for '--cluster'", err)
	}
	if err := cmd.Flags().MarkHidden("cluster"); err != nil {
		log.Fatalln("Failed to hide --cluster flag on registry create command")
	}

	cmd.Flags().StringVarP(&flags.Image, "image", "i", fmt.Sprintf("%s:%s", k3d.DefaultRegistryImageRepo, k3d.DefaultRegistryImageTag), "Specify image used for the registry")

	cmd.Flags().StringVarP(&ppFlags.Port, "port", "p", "random", "Select which port the registry should be listening on on your machine (localhost) (Format: `[HOST:]HOSTPORT`)\n - Example: `k3d registry create --port 0.0.0.0:5111`")

	// done
	return cmd
}

// parseCreateRegistryCmd parses the command input into variables required to create a registry
func parseCreateRegistryCmd(cmd *cobra.Command, args []string, flags *regCreateFlags, ppFlags *regCreatePreProcessedFlags) (*k3d.Registry, []*k3d.Cluster) {

	// --cluster
	clusters := []*k3d.Cluster{}
	for _, name := range ppFlags.Clusters {
		clusters = append(clusters,
			&k3d.Cluster{
				Name: name,
			},
		)
	}

	// --port
	exposePort, err := cliutil.ParsePortExposureSpec(ppFlags.Port, k3d.DefaultRegistryPort)
	if err != nil {
		log.Errorln("Failed to parse registry port")
		log.Fatalln(err)
	}

	// set the name for the registry node
	registryName := ""
	if len(args) > 0 {
		registryName = fmt.Sprintf("%s-%s", k3d.DefaultObjectNamePrefix, args[0])
	}

	return &k3d.Registry{Host: registryName, Image: flags.Image, ExposureOpts: *exposePort}, clusters
}
