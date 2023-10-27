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
package node

import (
	"github.com/k3d-io/k3d/v5/cmd/util"
	"github.com/k3d-io/k3d/v5/pkg/client"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/spf13/cobra"
)

type nodeDeleteFlags struct {
	All               bool
	IncludeRegistries bool
}

// NewCmdNodeDelete returns a new cobra command
func NewCmdNodeDelete() *cobra.Command {
	flags := nodeDeleteFlags{}

	// create new cobra command
	cmd := &cobra.Command{
		Use:               "delete (NAME | --all)",
		Short:             "Delete node(s).",
		Long:              `Delete node(s).`,
		ValidArgsFunction: util.ValidArgsAvailableNodes,
		Run: func(cmd *cobra.Command, args []string) {
			nodes := parseDeleteNodeCmd(cmd, args, &flags)
			nodeDeleteOpts := k3d.NodeDeleteOpts{SkipLBUpdate: flags.All} // do not update LB, if we're deleting all nodes anyway

			if len(nodes) == 0 {
				l.Log().Infoln("No nodes found")
			} else {
				for _, node := range nodes {
					if err := client.NodeDelete(cmd.Context(), runtimes.SelectedRuntime, node, nodeDeleteOpts); err != nil {
						l.Log().Fatalln(err)
					}
				}
				l.Log().Infof("Successfully deleted %d node(s)!", len(nodes))
			}
		},
	}

	// add subcommands

	// add flags
	cmd.Flags().BoolVarP(&flags.All, "all", "a", false, "Delete all existing nodes")
	cmd.Flags().BoolVarP(&flags.IncludeRegistries, "registries", "r", false, "Also delete registries")

	// done
	return cmd
}

// parseDeleteNodeCmd parses the command input into variables required to delete nodes
func parseDeleteNodeCmd(cmd *cobra.Command, args []string, flags *nodeDeleteFlags) []*k3d.Node {
	var nodes []*k3d.Node
	var err error

	// --all
	if flags.All {
		if !flags.IncludeRegistries {
			l.Log().Infoln("Didn't set '--registries', so won't delete registries.")
		}
		nodes, err = client.NodeList(cmd.Context(), runtimes.SelectedRuntime)
		if err != nil {
			l.Log().Fatalln(err)
		}
		include := k3d.ClusterInternalNodeRoles
		exclude := []k3d.Role{}
		if flags.IncludeRegistries {
			include = append(include, k3d.RegistryRole)
		}
		nodes = client.NodeFilterByRoles(nodes, include, exclude)
		return nodes
	}

	if !flags.All && len(args) < 1 {
		l.Log().Fatalln("Expecting at least one node name if `--all` is not set")
	}

	for _, name := range args {
		node, err := client.NodeGet(cmd.Context(), runtimes.SelectedRuntime, &k3d.Node{Name: name})
		if err != nil {
			l.Log().Fatalln(err)
		}
		nodes = append(nodes, node)
	}

	return nodes
}
