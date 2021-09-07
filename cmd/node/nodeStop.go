/*
Copyright © 2020-2021 The k3d Author(s)

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
	"github.com/rancher/k3d/v5/cmd/util"
	"github.com/rancher/k3d/v5/pkg/runtimes"
	"github.com/spf13/cobra"

	l "github.com/rancher/k3d/v5/pkg/logger"
	k3d "github.com/rancher/k3d/v5/pkg/types"
)

// NewCmdNodeStop returns a new cobra command
func NewCmdNodeStop() *cobra.Command {

	// create new command
	cmd := &cobra.Command{
		Use:               "stop NAME", // TODO: stopNode: allow one or more names or --all",
		Short:             "Stop an existing k3d node",
		Long:              `Stop an existing k3d node.`,
		ValidArgsFunction: util.ValidArgsAvailableNodes,
		Run: func(cmd *cobra.Command, args []string) {
			node := parseStopNodeCmd(cmd, args)
			if err := runtimes.SelectedRuntime.StopNode(cmd.Context(), node); err != nil {
				l.Log().Fatalln(err)
			}
		},
	}

	// done
	return cmd
}

// parseStopNodeCmd parses the command input into variables required to stop a node
func parseStopNodeCmd(cmd *cobra.Command, args []string) *k3d.Node {
	// node name // TODO: allow node filters, e.g. `k3d node stop mycluster@agent` to stop all agent nodes of cluster 'mycluster'
	if len(args) == 0 || len(args[0]) == 0 {
		l.Log().Fatalln("No node name given")
	}

	return &k3d.Node{Name: args[0]}
}
