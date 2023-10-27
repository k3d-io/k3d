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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/liggitt/tabwriter"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/k3d-io/k3d/v5/cmd/util"
	k3cluster "github.com/k3d-io/k3d/v5/pkg/client"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

// TODO : deal with --all flag to manage differentiate started cluster and stopped cluster like `docker ps` and `docker ps -a`
type clusterFlags struct {
	noHeader bool
	token    bool
	output   string
}

// NewCmdClusterList returns a new cobra command
func NewCmdClusterList() *cobra.Command {
	clusterFlags := clusterFlags{}

	// create new command
	cmd := &cobra.Command{
		Use:     "list [NAME [NAME...]]",
		Aliases: []string{"ls", "get"},
		Short:   "List cluster(s)",
		Long:    `List cluster(s).`,
		Run: func(cmd *cobra.Command, args []string) {
			clusters := buildClusterList(cmd.Context(), args)
			PrintClusters(clusters, clusterFlags)
		},
		ValidArgsFunction: util.ValidArgsAvailableClusters,
	}

	// add flags
	cmd.Flags().BoolVar(&clusterFlags.noHeader, "no-headers", false, "Disable headers")
	cmd.Flags().BoolVar(&clusterFlags.token, "token", false, "Print k3s cluster token")
	cmd.Flags().StringVarP(&clusterFlags.output, "output", "o", "", "Output format. One of: json|yaml")

	// add subcommands

	// done
	return cmd
}

func buildClusterList(ctx context.Context, args []string) []*k3d.Cluster {
	var clusters []*k3d.Cluster
	var err error

	if len(args) == 0 {
		// cluster name not specified : get all clusters
		clusters, err = k3cluster.ClusterList(ctx, runtimes.SelectedRuntime)
		if err != nil {
			l.Log().Fatalln(err)
		}
	} else {
		for _, clusterName := range args {
			// cluster name specified : get specific cluster
			retrievedCluster, err := k3cluster.ClusterGet(ctx, runtimes.SelectedRuntime, &k3d.Cluster{Name: clusterName})
			if err != nil {
				l.Log().Fatalln(err)
			}
			clusters = append(clusters, retrievedCluster)
		}
	}

	return clusters
}

// PrintPrintClusters : display list of cluster
func PrintClusters(clusters []*k3d.Cluster, flags clusterFlags) {
	// the output details printed when we dump JSON/YAML
	type jsonOutput struct {
		k3d.Cluster
		ServersRunning int  `json:"serversRunning"`
		ServersCount   int  `json:"serversCount"`
		AgentsRunning  int  `json:"agentsRunning"`
		AgentsCount    int  `json:"agentsCount"`
		LoadBalancer   bool `json:"hasLoadbalancer,omitempty"`
	}

	jsonOutputEntries := []jsonOutput{}

	outputFormat := strings.ToLower(flags.output)

	tabwriter := tabwriter.NewWriter(os.Stdout, 6, 4, 3, ' ', tabwriter.RememberWidths)
	defer tabwriter.Flush()

	if outputFormat != "json" && outputFormat != "yaml" {
		if !flags.noHeader {
			headers := []string{"NAME", "SERVERS", "AGENTS", "LOADBALANCER"} // TODO: getCluster: add status column
			if flags.token {
				headers = append(headers, "TOKEN")
			}
			_, err := fmt.Fprintf(tabwriter, "%s\n", strings.Join(headers, "\t"))
			if err != nil {
				l.Log().Fatalln("Failed to print headers")
			}
		}
	}

	k3cluster.SortClusters(clusters)

	for _, cluster := range clusters {
		serverCount, serversRunning := cluster.ServerCountRunning()
		agentCount, agentsRunning := cluster.AgentCountRunning()
		hasLB := cluster.HasLoadBalancer()

		if outputFormat == "json" || outputFormat == "yaml" {
			entry := jsonOutput{
				Cluster:        *cluster,
				ServersRunning: serversRunning,
				ServersCount:   serverCount,
				AgentsRunning:  agentsRunning,
				AgentsCount:    agentCount,
				LoadBalancer:   hasLB,
			}

			if !flags.token {
				entry.Token = ""
			}

			// clear some things
			entry.ExternalDatastore = nil

			jsonOutputEntries = append(jsonOutputEntries, entry)
		} else {
			if flags.token {
				fmt.Fprintf(tabwriter, "%s\t%d/%d\t%d/%d\t%t\t%s\n", cluster.Name, serversRunning, serverCount, agentsRunning, agentCount, hasLB, cluster.Token)
			} else {
				fmt.Fprintf(tabwriter, "%s\t%d/%d\t%d/%d\t%t\n", cluster.Name, serversRunning, serverCount, agentsRunning, agentCount, hasLB)
			}
		}
	}

	if outputFormat == "json" {
		b, err := json.Marshal(jsonOutputEntries)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(b))
	} else if outputFormat == "yaml" {
		b, err := yaml.Marshal(jsonOutputEntries)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(b))
	}
}
