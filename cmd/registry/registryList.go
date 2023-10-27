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
package registry

import (
	"fmt"
	"strings"

	"github.com/k3d-io/k3d/v5/cmd/util"
	"github.com/k3d-io/k3d/v5/pkg/client"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/liggitt/tabwriter"
	"github.com/spf13/cobra"
)

type registryListFlags struct {
	noHeader bool
	output   string
}

// NewCmdRegistryList creates a new cobra command
func NewCmdRegistryList() *cobra.Command {
	registryListFlags := registryListFlags{}

	// create new command
	cmd := &cobra.Command{
		Use:               "list [NAME [NAME...]]",
		Aliases:           []string{"ls", "get"},
		Short:             "List registries",
		Long:              `List registries.`,
		Args:              cobra.MinimumNArgs(0), // 0 or more; 0 = all
		ValidArgsFunction: util.ValidArgsAvailableRegistries,
		Run: func(cmd *cobra.Command, args []string) {
			var existingNodes []*k3d.Node

			nodes := []*k3d.Node{}
			for _, name := range args {
				nodes = append(nodes, &k3d.Node{
					Name: name,
				})
			}

			if len(nodes) == 0 { // Option a)  no name specified -> get all registries
				found, err := client.NodeList(cmd.Context(), runtimes.SelectedRuntime)
				if err != nil {
					l.Log().Fatalln(err)
				}
				existingNodes = append(existingNodes, found...)
			} else { // Option b) registry name(s) specified -> get specific registries
				for _, node := range nodes {
					l.Log().Tracef("Node %s", node.Name)
					found, err := client.NodeGet(cmd.Context(), runtimes.SelectedRuntime, node)
					if err != nil {
						l.Log().Fatalln(err)
					}
					existingNodes = append(existingNodes, found)
				}
			}
			existingNodes = client.NodeFilterByRoles(existingNodes, []k3d.Role{k3d.RegistryRole}, []k3d.Role{})

			// print existing registries
			headers := &[]string{}
			if !registryListFlags.noHeader {
				headers = &[]string{"NAME", "ROLE", "CLUSTER", "STATUS"}
			}

			util.PrintNodes(existingNodes, registryListFlags.output,
				headers, util.NodePrinterFunc(func(tabwriter *tabwriter.Writer, node *k3d.Node) {
					cluster := "*"
					if _, ok := node.RuntimeLabels[k3d.LabelClusterName]; ok {
						cluster = node.RuntimeLabels[k3d.LabelClusterName]
					}
					fmt.Fprintf(tabwriter, "%s\t%s\t%s\t%s\n",
						strings.TrimPrefix(node.Name, "/"),
						string(node.Role),
						cluster,
						node.State.Status,
					)
				}),
			)
		},
	}

	// add flags
	cmd.Flags().BoolVar(&registryListFlags.noHeader, "no-headers", false, "Disable headers")
	cmd.Flags().StringVarP(&registryListFlags.output, "output", "o", "", "Output format. One of: json|yaml")

	// add subcommands

	// done
	return cmd
}
