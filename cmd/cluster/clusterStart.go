/*
Copyright © 2020-2023 The k3d Author(s)

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
package cluster

import (
	"time"

	"github.com/k3d-io/k3d/v5/cmd/util"
	"github.com/k3d-io/k3d/v5/pkg/client"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	"github.com/spf13/cobra"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

// NewCmdClusterStart returns a new cobra command
func NewCmdClusterStart() *cobra.Command {
	startClusterOpts := k3d.ClusterStartOpts{
		Intent: k3d.IntentClusterStart,
	}

	// create new command
	cmd := &cobra.Command{
		Use:               "start [NAME [NAME...] | --all]",
		Long:              `Start existing k3d cluster(s)`,
		Short:             "Start existing k3d cluster(s)",
		ValidArgsFunction: util.ValidArgsAvailableClusters,
		Run: func(cmd *cobra.Command, args []string) {
			clusters := parseStartClusterCmd(cmd, args)
			if len(clusters) == 0 {
				l.Log().Infoln("No clusters found")
			} else {
				for _, c := range clusters {
					envInfo, err := client.GatherEnvironmentInfo(cmd.Context(), runtimes.SelectedRuntime, c)
					if err != nil {
						l.Log().Fatalf("failed to gather info about cluster environment: %v", err)
					}
					startClusterOpts.EnvironmentInfo = envInfo

					// Get pre-defined clusterStartOpts from cluster
					fetchedClusterStartOpts, err := client.GetClusterStartOptsFromLabels(c)
					if err != nil {
						l.Log().Fatalf("failed to get cluster start opts from cluster labels: %v", err)
					}

					// override only a few clusterStartOpts from fetched opts
					startClusterOpts.HostAliases = fetchedClusterStartOpts.HostAliases

					// start the cluster
					if err := client.ClusterStart(cmd.Context(), runtimes.SelectedRuntime, c, startClusterOpts); err != nil {
						l.Log().Fatalln(err)
					}
					l.Log().Infof("Started cluster '%s'", c.Name)
				}
			}
		},
	}

	// add flags
	cmd.Flags().BoolP("all", "a", false, "Start all existing clusters")
	cmd.Flags().BoolVar(&startClusterOpts.WaitForServer, "wait", true, "Wait for the server(s) (and loadbalancer) to be ready before returning.")
	cmd.Flags().DurationVar(&startClusterOpts.Timeout, "timeout", 0*time.Second, "Maximum waiting time for '--wait' before canceling/returning.")

	// add subcommands

	// done
	return cmd
}

// parseStartClusterCmd parses the command input into variables required to start clusters
func parseStartClusterCmd(cmd *cobra.Command, args []string) []*k3d.Cluster {
	// --all
	var clusters []*k3d.Cluster

	if all, err := cmd.Flags().GetBool("all"); err != nil {
		l.Log().Fatalln(err)
	} else if all {
		clusters, err = client.ClusterList(cmd.Context(), runtimes.SelectedRuntime)
		if err != nil {
			l.Log().Fatalln(err)
		}
		return clusters
	}

	clusternames := []string{k3d.DefaultClusterName}
	if len(args) != 0 {
		clusternames = args
	}

	for _, name := range clusternames {
		cluster, err := client.ClusterGet(cmd.Context(), runtimes.SelectedRuntime, &k3d.Cluster{Name: name})
		if err != nil {
			l.Log().Fatalln(err)
		}
		clusters = append(clusters, cluster)
	}

	return clusters
}
