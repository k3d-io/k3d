/*
Copyright © 2019 Thorsten Klein <iwilltry42@gmail.com>

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
package create

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rancher/k3d/pkg/cluster"
	"github.com/rancher/k3d/pkg/runtimes"
	k3d "github.com/rancher/k3d/pkg/types"
	log "github.com/sirupsen/logrus"
)

// NewCmdCreateNode returns a new cobra command
func NewCmdCreateNode() *cobra.Command {

	// create new command
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Create a new k3s node in docker",
		Long:  `Create a new containerized k3s node (k3s in docker).`,
		Args:  cobra.ExactArgs(1), // exactly one name accepted // TODO: if not specified, inherit from cluster that the node shall belong to, if that is specified
		Run: func(cmd *cobra.Command, args []string) {
			cluster.CreateNodes(parseCreateNodeCmd(cmd, args))
		},
	}

	// add flags
	cmd.Flags().Int("replicas", 1, "Number of replicas of this node specification.")
	cmd.Flags().String("role", "worker", "Specify node role [master, worker]")
	cmd.Flags().StringP("cluster", "c", "", "Select the cluster that the node shall connect to.")
	cmd.Flags().String("image", k3d.DefaultK3sImageRepo, "Specify k3s image used for the node(s)") // TODO: get image version tag

	// done
	return cmd
}

// parseCreateNodeCmd parses the command input into variables required to create a cluster
func parseCreateNodeCmd(cmd *cobra.Command, args []string) ([]*k3d.Node, runtimes.Runtime) {

	// --runtime
	rt, err := cmd.Flags().GetString("runtime")
	if err != nil {
		log.Fatalln("No runtime specified")
	}
	runtime, err := runtimes.GetRuntime(rt)
	if err != nil {
		log.Fatalln(err)
	}

	// --replicas
	replicas, err := cmd.Flags().GetInt("replicas")
	if err != nil {
		log.Errorln("No replica count specified")
		log.Fatalln(err)
	}

	// --role
	role, err := cmd.Flags().GetString("role")
	if err != nil {
		log.Errorln("No node role specified")
		log.Fatalln(err)
	}
	if _, ok := k3d.DefaultK3dRoles[role]; !ok {
		log.Fatalf("Unknown node role '%s'\n", role)
	}

	// --image
	image, err := cmd.Flags().GetString("image")
	if err != nil {
		log.Errorln("No image specified")
		log.Fatalln(err)
	}

	// generate list of nodes
	nodes := []*k3d.Node{}
	for i := 0; i < replicas; i++ {
		node := &k3d.Node{
			Name:  fmt.Sprintf("%s-%s-%d", k3d.DefaultObjectNamePrefix, args[0], i),
			Role:  role,
			Image: image,
		}
		nodes = append(nodes, node)
	}

	return nodes, runtime
}
