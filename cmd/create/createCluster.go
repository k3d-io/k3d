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

	cliutil "github.com/rancher/k3d/cmd/util"
	k3dCluster "github.com/rancher/k3d/pkg/cluster"
	"github.com/rancher/k3d/pkg/runtimes"
	k3d "github.com/rancher/k3d/pkg/types"

	log "github.com/sirupsen/logrus"
)

// NewCmdCreateCluster returns a new cobra command
func NewCmdCreateCluster() *cobra.Command {

	// create new command
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Create a new k3s cluster in docker",
		Long:  `Create a new k3s cluster with containerized nodes (k3s in docker).`,
		Args:  cobra.ExactArgs(1), // exactly one cluster name can be set // TODO: if not specified, use k3d.DefaultClusterName
		Run: func(cmd *cobra.Command, args []string) {
			runtime, cluster := parseCreateClusterCmd(cmd, args)
			if err := k3dCluster.CreateCluster(cluster, runtime); err != nil {
				log.Fatalln(err)
			}
		},
	}

	// add flags
	cmd.Flags().StringP("api-port", "a", "6443", "Specify the Kubernetes API server port (Format: `--api-port [host:]port`") // TODO: how to handle this for multi-master setups?
	cmd.Flags().IntP("masters", "m", 1, "Specify how many masters you want to create")
	cmd.Flags().IntP("workers", "w", 0, "Specify how many workers you want to create")
	cmd.Flags().String("config", "", "Specify a cluster configuration file")                                     // TODO: to implement
	cmd.Flags().String("image", k3d.DefaultK3sImageRepo, "Specify k3s image that you want to use for the nodes") // TODO: get image version
	cmd.Flags().String("network", "", "Join an existing network")
	cmd.Flags().String("secret", "", "Specify a cluster secret. By default, we generate one.")
	cmd.Flags().StringArrayP("volume", "v", nil, "Mount volumes into the nodes (Format: `--volume [SOURCE:]DEST[@SELECTOR[;SELECTOR...]]`")

	// add subcommands

	// done
	return cmd
}

// parseCreateClusterCmd parses the command input into variables required to create a cluster
func parseCreateClusterCmd(cmd *cobra.Command, args []string) (runtimes.Runtime, *k3d.Cluster) {
	// --runtime
	rt, err := cmd.Flags().GetString("runtime")
	if err != nil {
		log.Fatalln("No runtime specified")
	}
	runtime, err := runtimes.GetRuntime(rt)
	if err != nil {
		log.Fatalln(err)
	}

	// --image
	image, err := cmd.Flags().GetString("image")
	if err != nil {
		log.Errorln("No image specified")
		log.Fatalln(err)
	}

	// --masters
	masterCount, err := cmd.Flags().GetInt("masters")
	if err != nil {
		log.Fatalln(err)
	}

	// TODO: allow more than one master
	if masterCount > 1 {
		log.Fatalln("Only one master node supported right now!")
	}

	// --workers
	workerCount, err := cmd.Flags().GetInt("workers")
	if err != nil {
		log.Fatalln(err)
	}

	// --network
	network, err := cmd.Flags().GetString("network")
	if err != nil {
		log.Fatalln(err)
	}

	// --secret
	secret, err := cmd.Flags().GetString("secret")
	if err != nil {
		log.Fatalln(err)
	}

	// --api-port
	apiPort, err := cmd.Flags().GetString("api-port")
	if err != nil {
		log.Fatalln(err)
	}
	exposeAPI, err := cliutil.ParseAPIPort(apiPort)
	if err != nil {
		log.Fatalln(err)
	}

	// --volume
	volumeFlags, err := cmd.Flags().GetStringArray("volume")
	if err != nil {
		log.Fatalln(err)
	}
	volumeFilterMap := make(map[string]string, 1)
	for _, volumeFlag := range volumeFlags {
		log.Debugf("Parsing vol flag %+v", volumeFlag)
		volume, filter, err := cliutil.SplitFilterFromFlag(volumeFlag)
		if err != nil {
			log.Fatalln(err)
		}
		log.Debugf("Parsed vol flag %+v + filter %+v", volume, filter)
		volume, err = cliutil.ValidateVolumeMount(volume)
		if err != nil {
			log.Fatalln(err)
		}

		// create new entry or append filter to existing entry
		if _, exists := volumeFilterMap[volume]; exists {
			volumeFilterMap[volume] = fmt.Sprintf("%s;%s", volumeFilterMap[volume], filter)
		} else {
			volumeFilterMap[volume] = filter
		}
		log.Debugf("volFilterMap %+v", volumeFilterMap)
	}
	log.Debugf("volumeFIlterMap: %+v", volumeFilterMap)

	/* generate cluster */
	cluster := &k3d.Cluster{
		Name:    args[0], // TODO: validate name0
		Network: network,
		Secret:  secret,
	}

	// generate list of nodes
	cluster.Nodes = []*k3d.Node{}

	// -> master nodes
	for i := 0; i < masterCount; i++ {
		node := k3d.Node{
			Role:       k3d.MasterRole,
			Image:      image,
			MasterOpts: k3d.MasterOpts{},
		}

		// expose API Port
		if i == 0 { // TODO:
			node.MasterOpts.ExposeAPI = exposeAPI
		}

		// append node to list
		cluster.Nodes = append(cluster.Nodes, &node)
	}

	// -> worker nodes
	for i := 0; i < workerCount; i++ {
		node := k3d.Node{
			Role:  k3d.WorkerRole,
			Image: image,
		}

		cluster.Nodes = append(cluster.Nodes, &node)
	}

	// append volumes
	// TODO:
	for volume, filter := range volumeFilterMap {
		nodes, err := cliutil.FilterNodes(cluster.Nodes, filter)
		if err != nil {
			log.Fatalln(err)
		}
		for _, node := range nodes {
			node.Volumes = append(node.Volumes, volume)
		}
	}

	return runtime, cluster
}
