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
	"os"
	"sort"
	"strings"

	"github.com/liggitt/tabwriter"
	"github.com/rancher/k3d/v3/cmd/util"
	"github.com/rancher/k3d/v3/pkg/cluster"
	"github.com/rancher/k3d/v3/pkg/runtimes"
	k3d "github.com/rancher/k3d/v3/pkg/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewCmdRegistryList creates a new cobra command
func NewCmdRegistryList() *cobra.Command {
	// create new command
	cmd := &cobra.Command{
		Use:               "list [NAME [NAME...]]",
		Aliases:           []string{"ls", "get"},
		Short:             "List registries",
		Long:              `List registries.`,
		Args:              cobra.MinimumNArgs(0), // 0 or more; 0 = all
		ValidArgsFunction: util.ValidArgsAvailableRegistries,
		Run: func(cmd *cobra.Command, args []string) {
			nodes, headersOff := parseRegistryListCmd(cmd, args)
			var existingNodes []*k3d.Node
			if len(nodes) == 0 { // Option a)  no name specified -> get all registries
				found, err := cluster.NodeList(cmd.Context(), runtimes.SelectedRuntime)
				if err != nil {
					log.Fatalln(err)
				}
				existingNodes = append(existingNodes, found...)
			} else { // Option b) registry name(s) specified -> get specific registries
				for _, node := range nodes {
					log.Debugf("Bla %s", node.Name)
					found, err := cluster.NodeGet(cmd.Context(), runtimes.SelectedRuntime, node)
					if err != nil {
						log.Fatalln(err)
					}
					existingNodes = append(existingNodes, found)
				}
			}
			existingNodes = cluster.NodeFilterByRoles(existingNodes, []k3d.Role{k3d.RegistryRole}, []k3d.Role{})
			// print existing registries
			if len(existingNodes) > 0 {
				printNodes(existingNodes, headersOff)
			}
		},
	}

	// add flags
	cmd.Flags().Bool("no-headers", false, "Disable headers")

	// add subcommands

	// done
	return cmd
}

func parseRegistryListCmd(cmd *cobra.Command, args []string) ([]*k3d.Node, bool) {
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
		nodes = append(nodes, &k3d.Node{
			Name: name,
		})
	}

	return nodes, headersOff
}

func printNodes(nodes []*k3d.Node, headersOff bool) {

	tabwriter := tabwriter.NewWriter(os.Stdout, 6, 4, 3, ' ', tabwriter.RememberWidths)
	defer tabwriter.Flush()

	if !headersOff {
		headers := []string{"NAME", "ROLE", "CLUSTER"} // TODO: add status
		_, err := fmt.Fprintf(tabwriter, "%s\n", strings.Join(headers, "\t"))
		if err != nil {
			log.Fatalln("Failed to print headers")
		}
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})

	for _, node := range nodes {
		fmt.Fprintf(tabwriter, "%s\t%s\t%s\n", strings.TrimPrefix(node.Name, "/"), string(node.Role), node.Labels[k3d.LabelClusterName])
	}
}
