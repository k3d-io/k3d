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
package create

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rancher/k3d/cmd/util"
	k3dc "github.com/rancher/k3d/pkg/cluster"
	"github.com/rancher/k3d/pkg/runtimes"
	k3d "github.com/rancher/k3d/pkg/types"
	log "github.com/sirupsen/logrus"
)

// NewCmdCreateRegistry returns a new cobra command
func NewCmdCreateRegistry() *cobra.Command {

	createRegistryOpts := k3d.CreateRegistryOpts{}

	// create new command
	cmd := &cobra.Command{
		Use:   "registry NAME",
		Short: "Create a new image registry",
		Long:  `Create a new containerized image registry.`,
		Args:  cobra.ExactArgs(1), // exactly one name accepted // TODO: if not specified, inherit from cluster that the node shall belong to, if that is specified
		Run: func(cmd *cobra.Command, args []string) {
			registryNode, clusters := parseCreateRegistryCmd(cmd, args)
			// TODO: add registryNode to cluster network
			// TODO: register registry in cluster's registries.yaml
		},
	}

	// add flags
	cmd.Flags().StringSliceVarP("cluster", "c", []string{}, "Select the cluster(s) that the registry shall connect to.")
	if err := cmd.RegisterFlagCompletionFunc("cluster", util.ValidArgsAvailableClusters); err != nil {
		log.Fatalln("Failed to register flag completion for '--cluster'", err)
	}

	cmd.Flags().StringP("image", "i", fmt.Sprintf("%s:%s", k3d.DefaultRegistryImageRepo, k3d.DefaultRegistryImageTag), "Specify registry image")

	// done
	return cmd
}

// parseCreateRegistryCmd parses the command input into variables required to create a cluster
func parseCreateRegistryCmd(cmd *cobra.Command, args []string) (*k3d.Node, []*k3d.Cluster) {

	// --image
	image, err := cmd.Flags().GetString("image")
	if err != nil {
		log.Errorln("No image specified")
		log.Fatalln(err)
	}

	// --cluster, -c
	clusterNames, err := cmd.Flags().GetStringSlice("cluster")
	if err != nil {
		log.Fatalln(err)
	}
	// generate list of clusters
	var attachClusters []*k3d.Cluster
	for _, clusterName := range clusterNames {
		cluster, err := k3dc.GetCluster(cmd.Context(), runtimes.SelectedRuntime, &k3d.Cluster{Name: clusterName})
		if err != nil {
			log.Errorln(err)
			log.Fatalf("Non-existent cluster '%s' specified", clusterName)
		}
		attachClusters = append(attachClusters, cluster)
	}

	// describe registry as a k3d node
	registryNode := &k3d.Node{
		Name:  fmt.Sprintf("%s-registry-%s", k3d.DefaultObjectNamePrefix, args[0]),
		Role:  k3d.NoRole,
		Image: image,
		Labels: map[string]string{
			k3d.LabelRole: string(k3d.NoRole),
		},
	}

	return registryNode, attachClusters
}
