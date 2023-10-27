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
	"github.com/docker/go-connections/nat"
	"github.com/k3d-io/k3d/v5/cmd/util"
	"github.com/k3d-io/k3d/v5/pkg/client"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/spf13/cobra"
)

// NewCmdNodeEdit returns a new cobra command
func NewCmdNodeEdit() *cobra.Command {
	// create new cobra command
	cmd := &cobra.Command{
		Use:               "edit NODE",
		Short:             "[EXPERIMENTAL] Edit node(s).",
		Long:              `[EXPERIMENTAL] Edit node(s).`,
		Args:              cobra.ExactArgs(1),
		Aliases:           []string{"update"},
		ValidArgsFunction: util.ValidArgsAvailableNodes,
		Run: func(cmd *cobra.Command, args []string) {
			existingNode, changeset := parseEditNodeCmd(cmd, args)

			l.Log().Debugf("===== Current =====\n%+v\n===== Changeset =====\n%+v\n", existingNode, changeset)

			if err := client.NodeEdit(cmd.Context(), runtimes.SelectedRuntime, existingNode, changeset); err != nil {
				l.Log().Fatalln(err)
			}

			l.Log().Infof("Successfully updated %s", existingNode.Name)
		},
	}

	// add subcommands

	// add flags
	cmd.Flags().StringArray("port-add", nil, "[EXPERIMENTAL] (serverlb only!) Map ports from the node container to the host (Format: `[HOST:][HOSTPORT:]CONTAINERPORT[/PROTOCOL][@NODEFILTER]`)\n - Example: `k3d node edit k3d-mycluster-serverlb --port-add 8080:80`")

	// done
	return cmd
}

// parseEditNodeCmd parses the command input into variables required to delete nodes
func parseEditNodeCmd(cmd *cobra.Command, args []string) (*k3d.Node, *k3d.Node) {
	existingNode, err := client.NodeGet(cmd.Context(), runtimes.SelectedRuntime, &k3d.Node{Name: args[0]})
	if err != nil {
		l.Log().Fatalln(err)
	}

	if existingNode == nil {
		l.Log().Infof("Node %s not found", args[0])
		return nil, nil
	}

	if existingNode.Role != k3d.LoadBalancerRole {
		l.Log().Fatalln("Currently only the loadbalancer can be updated!")
	}

	changeset := &k3d.Node{}

	/*
	 * --port-add
	 */
	portFlags, err := cmd.Flags().GetStringArray("port-add")
	if err != nil {
		l.Log().Errorln(err)
		return nil, nil
	}

	// init portmap
	changeset.Ports = nat.PortMap{}

	for _, flag := range portFlags {
		portmappings, err := nat.ParsePortSpec(flag)
		if err != nil {
			l.Log().Fatalf("Failed to parse port spec '%s': %+v", flag, err)
		}

		for _, pm := range portmappings {
			changeset.Ports[pm.Port] = append(changeset.Ports[pm.Port], pm.Binding)
		}
	}

	return existingNode, changeset
}
