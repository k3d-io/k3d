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
package node

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/liggitt/tabwriter"
	"github.com/rancher/k3d/v4/cmd/util"
	"github.com/rancher/k3d/v4/pkg/cluster"
	"github.com/rancher/k3d/v4/pkg/runtimes"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"
)

// NewCmdNodeList returns a new cobra command
func NewCmdNodeList() *cobra.Command {

	// create new command
	cmd := &cobra.Command{
		Use:               "list [NAME [NAME...]]",
		Aliases:           []string{"ls", "get"},
		Short:             "List node(s)",
		Long:              `List node(s).`,
		Args:              cobra.MinimumNArgs(0), // 0 or more; 0 = all
		ValidArgsFunction: util.ValidArgsAvailableNodes,
		Run: func(cmd *cobra.Command, args []string) {
			nodes, headersOff := parseGetNodeCmd(cmd, args)
			var existingNodes []*k3d.Node
			if len(nodes) == 0 { // Option a)  no name specified -> get all nodes
				found, err := cluster.NodeList(cmd.Context(), runtimes.SelectedRuntime)
				if err != nil {
					log.Fatalln(err)
				}
				existingNodes = append(existingNodes, found...)
			} else { // Option b) cluster name specified -> get specific cluster
				for _, node := range nodes {
					found, err := cluster.NodeGet(cmd.Context(), runtimes.SelectedRuntime, node)
					if err != nil {
						log.Fatalln(err)
					}
					existingNodes = append(existingNodes, found)
				}
			}
			// print existing clusters
			printNodes(existingNodes, headersOff)
		},
	}

	// add flags
	cmd.Flags().Bool("no-headers", false, "Disable headers")

	// add subcommands

	// done
	return cmd
}

func parseGetNodeCmd(cmd *cobra.Command, args []string) ([]*k3d.Node, bool) {
	// --no-headers
	headersOff, err := cmd.Flags().GetBool("no-headers")
	if err != nil {
		log.Fatalln(err)
	}

	// Args = node name
	if len(args) == 0 {
		return nil, headersOff
	}

	nodes := []*k3d.Node{}
	for _, name := range args {
		nodes = append(nodes, &k3d.Node{Name: name})
	}

	return nodes, headersOff
}

func printNodes(nodes []*k3d.Node, headersOff bool) {

	tabwriter := tabwriter.NewWriter(os.Stdout, 6, 4, 3, ' ', tabwriter.RememberWidths)
	defer tabwriter.Flush()

	if !headersOff {
		headers := []string{"NAME", "ROLE", "CLUSTER", "STATUS"}
		_, err := fmt.Fprintf(tabwriter, "%s\n", strings.Join(headers, "\t"))
		if err != nil {
			log.Fatalln("Failed to print headers")
		}
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})

	for _, node := range nodes {
		fmt.Fprintf(tabwriter, "%s\t%s\t%s\t%s\n", strings.TrimPrefix(node.Name, "/"), string(node.Role), node.Labels[k3d.LabelClusterName], node.State.Status)
	}
}
