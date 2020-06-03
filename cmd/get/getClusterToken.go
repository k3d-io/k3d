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
	"sort"
	"strings"

	"github.com/liggitt/tabwriter"
	cliutil "github.com/rancher/k3d/cmd/util"
	"github.com/rancher/k3d/pkg/cluster"
	"github.com/rancher/k3d/pkg/runtimes"
	k3d "github.com/rancher/k3d/pkg/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type clusterTokenFlags struct {
	validableFlags cliutil.ValidableFlags
	noHeader       bool
}

// NewCmdGetClusterToken returns a new cobra command
func NewCmdGetClusterToken() *cobra.Command {

	getClusterTokenFlags := clusterTokenFlags{}

	// create new command
	cmd := &cobra.Command{
		Use:   "k3stoken [CLUSTER [CLUSTER [...]] | --all]",
		Short: "Get cluster token",
		Long:  `Get k3s cluster token.`,
		Args: func(cmd *cobra.Command, args []string) error {
			return cliutil.ValidateClusterNameOrAllFlag(args, getClusterTokenFlags.validableFlags)
		},
		Run: func(cmd *cobra.Command, args []string) {
			var clusters []*k3d.Cluster
			var err error

			// generate list of clusters
			if getClusterTokenFlags.validableFlags.All {
				clusters, err = cluster.GetClusters(runtimes.SelectedRuntime)
				if err != nil {
					log.Fatalln(err)
				}
			} else {
				for _, clusterName := range args {
					retrievedCluster, err := cluster.GetCluster(&k3d.Cluster{Name: clusterName}, runtimes.SelectedRuntime)
					if err != nil {
						log.Fatalln(err)
					}
					clusters = append(clusters, retrievedCluster)
				}
			}

			// pretty print token
			printToken(clusters, getClusterTokenFlags.noHeader)
		},
	}

	// add flags
	cmd.Flags().BoolVarP(&getClusterTokenFlags.validableFlags.All, "all", "a", false, "Get k3s token from all existing clusters")
	cmd.Flags().BoolVar(&getClusterTokenFlags.noHeader, "no-headers", false, "Disable headers")

	// done
	return cmd
}

func printToken(clusters []*k3d.Cluster, headersOff bool) {

	tabwriter := tabwriter.NewWriter(os.Stdout, 6, 4, 3, ' ', tabwriter.RememberWidths)
	defer tabwriter.Flush()

	if !headersOff {
		headers := []string{"CLUSTER", "TOKEN"}
		_, err := fmt.Fprintf(tabwriter, "%s\n", strings.Join(headers, "\t"))
		if err != nil {
			log.Fatalln("Failed to print headers")
		}
	}

	// alphabetical sort by cluster name
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Name < clusters[j].Name
	})

	for _, cluster := range clusters {
		fmt.Fprintf(tabwriter, "%s\t%s\n", cluster.Name, string(cluster.Token))
	}
}
