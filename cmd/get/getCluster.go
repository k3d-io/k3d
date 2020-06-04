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
package get

import (
	"fmt"
	"os"
	"strings"

	k3dutil "github.com/rancher/k3d/cmd/util"
	k3dcluster "github.com/rancher/k3d/pkg/cluster"
	k3d "github.com/rancher/k3d/pkg/types"
	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"

	"github.com/liggitt/tabwriter"
)

// TODO : deal with --all flag to manage differentiate started cluster and stopped cluster like `docker ps` and `docker ps -a`
type clusterFlags struct {
	noHeader bool
	token    bool
}

// NewCmdGetCluster returns a new cobra command
func NewCmdGetCluster() *cobra.Command {

	clusterFlags := clusterFlags{}

	// create new command
	cmd := &cobra.Command{
		Use:     "cluster [NAME [NAME...]]",
		Aliases: []string{"clusters"},
		Short:   "Get cluster(s)",
		Long:    `Get cluster(s).`,
		Run: func(cmd *cobra.Command, args []string) {
			clusters := k3dutil.BuildClusterList(cmd.Context(), args)
			PrintClusters(clusters, clusterFlags)
		},
	}

	// add flags
	cmd.Flags().BoolVar(&clusterFlags.noHeader, "no-headers", false, "Disable headers")
	cmd.Flags().BoolVar(&clusterFlags.token, "token", false, "Print k3s cluster token")

	// add subcommands

	// done
	return cmd
}

// PrintPrintClusters : display list of cluster
func PrintClusters(clusters []*k3d.Cluster, flags clusterFlags) {

	tabwriter := tabwriter.NewWriter(os.Stdout, 6, 4, 3, ' ', tabwriter.RememberWidths)
	defer tabwriter.Flush()

	if !flags.noHeader {
		headers := []string{"NAME", "MASTERS", "WORKERS"} // TODO: getCluster: add status column
		if flags.token {
			headers = append(headers, "TOKEN")
		}
		_, err := fmt.Fprintf(tabwriter, "%s\n", strings.Join(headers, "\t"))
		if err != nil {
			log.Fatalln("Failed to print headers")
		}
	}

	k3dcluster.SortClusters(clusters)

	for _, cluster := range clusters {
		masterCount := cluster.MasterCount()
		workerCount := cluster.WorkerCount()

		if flags.token {
			fmt.Fprintf(tabwriter, "%s\t%d\t%d\t%s\n", cluster.Name, masterCount, workerCount, cluster.Token)
		} else {
			fmt.Fprintf(tabwriter, "%s\t%d\t%d\n", cluster.Name, masterCount, workerCount)
		}
	}
}
