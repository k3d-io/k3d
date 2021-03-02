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
	"time"

	"github.com/spf13/cobra"

	"github.com/rancher/k3d/v4/cmd/util"
	k3dc "github.com/rancher/k3d/v4/pkg/client"
	"github.com/rancher/k3d/v4/pkg/runtimes"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	"github.com/rancher/k3d/v4/version"
	log "github.com/sirupsen/logrus"
)

// NewCmdNodeCreate returns a new cobra command
func NewCmdNodeCreate() *cobra.Command {

	createNodeOpts := k3d.NodeCreateOpts{}

	// create new command
	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a new k3s node in docker",
		Long:  `Create a new containerized k3s node (k3s in docker).`,
		Args:  cobra.ExactArgs(1), // exactly one name accepted // TODO: if not specified, inherit from cluster that the node shall belong to, if that is specified
		Run: func(cmd *cobra.Command, args []string) {
			nodes, cluster := parseCreateNodeCmd(cmd, args)
			if err := k3dc.NodeAddToClusterMulti(cmd.Context(), runtimes.SelectedRuntime, nodes, cluster, createNodeOpts); err != nil {
				log.Errorf("Failed to add nodes to cluster '%s'", cluster.Name)
				log.Fatalln(err)
			}
		},
	}

	// add flags
	cmd.Flags().Int("replicas", 1, "Number of replicas of this node specification.")
	cmd.Flags().String("role", string(k3d.AgentRole), "Specify node role [server, agent]")
	if err := cmd.RegisterFlagCompletionFunc("role", util.ValidArgsNodeRoles); err != nil {
		log.Fatalln("Failed to register flag completion for '--role'", err)
	}
	cmd.Flags().StringP("cluster", "c", k3d.DefaultClusterName, "Select the cluster that the node shall connect to.")
	if err := cmd.RegisterFlagCompletionFunc("cluster", util.ValidArgsAvailableClusters); err != nil {
		log.Fatalln("Failed to register flag completion for '--cluster'", err)
	}

	cmd.Flags().StringP("image", "i", fmt.Sprintf("%s:%s", k3d.DefaultK3sImageRepo, version.GetK3sVersion(false)), "Specify k3s image used for the node(s)")

	cmd.Flags().BoolVar(&createNodeOpts.Wait, "wait", false, "Wait for the node(s) to be ready before returning.")
	cmd.Flags().DurationVar(&createNodeOpts.Timeout, "timeout", 0*time.Second, "Maximum waiting time for '--wait' before canceling/returning.")

	// done
	return cmd
}

// parseCreateNodeCmd parses the command input into variables required to create a cluster
func parseCreateNodeCmd(cmd *cobra.Command, args []string) ([]*k3d.Node, *k3d.Cluster) {

	// --replicas
	replicas, err := cmd.Flags().GetInt("replicas")
	if err != nil {
		log.Errorln("No replica count specified")
		log.Fatalln(err)
	}

	// --role
	roleStr, err := cmd.Flags().GetString("role")
	if err != nil {
		log.Errorln("No node role specified")
		log.Fatalln(err)
	}
	if _, ok := k3d.NodeRoles[roleStr]; !ok {
		log.Fatalf("Unknown node role '%s'\n", roleStr)
	}
	role := k3d.NodeRoles[roleStr]

	// --image
	image, err := cmd.Flags().GetString("image")
	if err != nil {
		log.Errorln("No image specified")
		log.Fatalln(err)
	}

	// --cluster
	clusterName, err := cmd.Flags().GetString("cluster")
	if err != nil {
		log.Fatalln(err)
	}
	cluster := &k3d.Cluster{
		Name: clusterName,
	}

	// generate list of nodes
	nodes := []*k3d.Node{}
	for i := 0; i < replicas; i++ {
		node := &k3d.Node{
			Name:  fmt.Sprintf("%s-%s-%d", k3d.DefaultObjectNamePrefix, args[0], i),
			Role:  role,
			Image: image,
			Labels: map[string]string{
				k3d.LabelRole: roleStr,
			},
			Restart: true,
		}
		nodes = append(nodes, node)
	}

	return nodes, cluster
}
