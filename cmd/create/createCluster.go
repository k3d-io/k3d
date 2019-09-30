/*
Copyright © 2019 Thorsten Klein <iwilltry42@gmail.com>

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
package create

import (
	"github.com/spf13/cobra"

	k3dCluster "github.com/rancher/k3d/pkg/cluster"
	"github.com/rancher/k3d/pkg/runtimes"
	k3d "github.com/rancher/k3d/pkg/types"

	log "github.com/sirupsen/logrus"
)

// NewCmdCreateCluster returns a new cobra command
func NewCmdCreateCluster() *cobra.Command {

	// create new command
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Create a new k3s cluster in docker",
		Long:  `Create a new k3s cluster with containerized nodes (k3s in docker).`,
		Args:  cobra.ExactArgs(1), // exactly one cluster name can be set // TODO: if not specified, use k3d.DefaultClusterName
		Run: func(cmd *cobra.Command, args []string) {
			runtime, cluster := parseCreateClusterCmd(cmd, args)
			if err := k3dCluster.CreateCluster(cluster, runtime); err != nil {
				log.Fatalln(err)
			}
		},
	}

	// add flags
	cmd.Flags().StringP("api-port", "a", "6443", "Specify the Kubernetes API server port (Format: `--api-port [host:]port`")
	cmd.Flags().IntP("masters", "m", 1, "Specify how many masters you want to create")
	cmd.Flags().IntP("workers", "w", 0, "Specify how many workers you want to create")
	cmd.Flags().String("config", "", "Specify a cluster configuration file") // TODO: to implement

	// add subcommands

	// done
	return cmd
}

// parseCreateClusterCmd parses the command input into variables required to create a cluster
func parseCreateClusterCmd(cmd *cobra.Command, args []string) (runtimes.Runtime, *k3d.Cluster) {
	rt, err := cmd.Flags().GetString("runtime")
	if err != nil {
		log.Fatalln("Runtime not defined")
	}
	runtime, err := runtimes.GetRuntime(rt)
	if err != nil {
		log.Fatalln(err)
	}
	cluster := k3d.Cluster{Name: args[0]}

	return runtime, &cluster
}
