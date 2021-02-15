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
	log "github.com/sirupsen/logrus"

	"github.com/rancher/k3d/v4/pkg/runtimes"
	k3d "github.com/rancher/k3d/v4/pkg/types"

	"github.com/rancher/k3d/v4/pkg/client"

	"github.com/spf13/cobra"
)

type regStartFlags struct {
	All bool
}

// NewCmdRegistryStart creates a new cobra command
func NewCmdRegistryStart() *cobra.Command {
	flags := &regStartFlags{}

	// create new command
	cmd := &cobra.Command{
		Use:   "start (NAME | --all)",
		Short: "Start a registry/registries",
		Long:  `Start a registry/registries`,
		Args:  cobra.MaximumNArgs(1), // maximum one name accepted
		Run: func(cmd *cobra.Command, args []string) {
			nodes := parseRegistryStartCmd(cmd, args, flags)
			if len(nodes) == 0 {
				log.Infoln("No registries found")
			} else {
				for _, node := range nodes {
					err := client.RegistryStart(cmd.Context(), runtimes.SelectedRuntime, node)
					if err != nil {
						log.Fatalln(err)
					}
				}
			}
		},
	}

	// add flags
	cmd.Flags().BoolVarP(&flags.All, "all", "a", false, "Start all stopped registries")

	// done
	return cmd
}

// parseRegistryStartCmd parses the command input into variables required to start nodes
func parseRegistryStartCmd(cmd *cobra.Command, args []string, flags *regStartFlags) []*k3d.Node {

	var nodes []*k3d.Node
	var err error

	if flags.All {
		nodes, err = client.NodeList(cmd.Context(), runtimes.SelectedRuntime)
		if err != nil {
			log.Fatalln(err)
		}
	}

	if !flags.All && len(args) < 1 {
		log.Fatalln("Expecting at least one registry name if `--all` is not set")
	}

	for _, name := range args {
		node, err := client.NodeGet(cmd.Context(), runtimes.SelectedRuntime, &k3d.Node{Name: name, State: k3d.NodeState{Running: false}})
		if err != nil {
			log.Fatalln(err)
		}
		nodes = append(nodes, node)
	}

	nodes = client.NodeFilterByRoles(nodes, []k3d.Role{k3d.RegistryRole}, []k3d.Role{})

	return nodes
}
