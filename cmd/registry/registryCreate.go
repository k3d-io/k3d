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

	"github.com/rancher/k3d/v3/pkg/runtimes"
	k3d "github.com/rancher/k3d/v3/pkg/types"

	k3dcluster "github.com/rancher/k3d/v3/pkg/cluster"

	cliutil "github.com/rancher/k3d/v3/cmd/util"
	"github.com/spf13/cobra"
)

// NewCmdRegistryCreate returns a new cobra command
func NewCmdRegistryCreate() *cobra.Command {

	// create new command
	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a new registry",
		Long:  `Create a new registry.`,
		Args:  cobra.MaximumNArgs(1), // maximum one name accepted
		Run: func(cmd *cobra.Command, args []string) {
			name, image, port, clusters := parseCreateRegistryCmd(cmd, args)
			if err := k3dcluster.RegistryCreate(cmd.Context(), runtimes.SelectedRuntime, name, image, port, clusters); err != nil {
				log.Fatalln(err)
			}
		},
	}

	// add flags
	cmd.Flags().StringSliceP("cluster", "c", []string{k3d.DefaultClusterName}, "Select the cluster(s) that the registry shall connect to.")
	if err := cmd.RegisterFlagCompletionFunc("cluster", cliutil.ValidArgsAvailableClusters); err != nil {
		log.Fatalln("Failed to register flag completion for '--cluster'", err)
	}

	cmd.Flags().StringP("image", "i", fmt.Sprintf("%s:%s", k3d.DefaultRegistryImageRepo, k3d.DefaultRegistryImageTag), "Specify image used for the registry")

	cmd.Flags().StringP("port", "p", "random", "Select which port the registry should be listening on on your machine (localhost) (Format: `[HOST:]HOSTPORT`)\n - Example: `k3d registry create --port 0.0.0.0:5111`")

	// done
	return cmd
}

// parseCreateRegistryCmd parses the command input into variables required to create a registry
func parseCreateRegistryCmd(cmd *cobra.Command, args []string) (string, string, k3d.ExposePort, []*k3d.Cluster) {

	// --image
	image, err := cmd.Flags().GetString("image")
	if err != nil {
		log.Errorln("No image specified")
		log.Fatalln(err)
	}

	// --cluster
	clusters := []*k3d.Cluster{}
	clusterNames, err := cmd.Flags().GetStringSlice("cluster")
	if err != nil {
		log.Fatalln(err)
	}
	for _, name := range clusterNames {
		clusters = append(clusters,
			&k3d.Cluster{
				Name: name,
			},
		)
	}

	// --port
	port, err := cmd.Flags().GetString("port")
	if err != nil {
		log.Fatalln(err)
	}
	// port
	exposePort, err := cliutil.ParseExposePort(port)
	if err != nil {
		log.Errorln("Failed to parse registry port")
		log.Fatalln(err)
	}

	// set the name for the registry node
	registryName := ""
	if len(args) > 0 {
		registryName = fmt.Sprintf("%s-%s", k3d.DefaultObjectNamePrefix, args[0])
	}

	return registryName, image, exposePort, clusters
}
