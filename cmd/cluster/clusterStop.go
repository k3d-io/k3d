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
	"github.com/spf13/cobra"

	"github.com/k3d-io/k3d/v5/cmd/util"
	"github.com/k3d-io/k3d/v5/pkg/client"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

// NewCmdClusterStop returns a new cobra command
func NewCmdClusterStop() *cobra.Command {
	// create new command
	cmd := &cobra.Command{
		Use:               "stop [NAME [NAME...] | --all]",
		Short:             "Stop existing k3d cluster(s)",
		Long:              `Stop existing k3d cluster(s).`,
		ValidArgsFunction: util.ValidArgsAvailableClusters,
		Run: func(cmd *cobra.Command, args []string) {
			clusters := parseStopClusterCmd(cmd, args)
			if len(clusters) == 0 {
				l.Log().Infoln("No clusters found")
			} else {
				for _, c := range clusters {
					if err := client.ClusterStop(cmd.Context(), runtimes.SelectedRuntime, c); err != nil {
						l.Log().Fatalln(err)
					}
				}
			}
		},
	}

	// add flags
	cmd.Flags().BoolP("all", "a", false, "Stop all existing clusters")

	// add subcommands

	// done
	return cmd
}

// parseStopClusterCmd parses the command input into variables required to start clusters
func parseStopClusterCmd(cmd *cobra.Command, args []string) []*k3d.Cluster {
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
