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
package cluster

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	cliutil "github.com/rancher/k3d/v3/cmd/util"
	"github.com/rancher/k3d/v3/pkg/cluster"
	k3dCluster "github.com/rancher/k3d/v3/pkg/cluster"
	"github.com/rancher/k3d/v3/pkg/runtimes"
	k3d "github.com/rancher/k3d/v3/pkg/types"
	"github.com/rancher/k3d/v3/version"

	log "github.com/sirupsen/logrus"
)

const clusterCreateDescription = `
Create a new k3s cluster with containerized nodes (k3s in docker).
Every cluster will consist of one or more containers:
	- 1 (or more) server node container (k3s)
	- (optionally) 1 loadbalancer container as the entrypoint to the cluster (nginx)
	- (optionally) 1 (or more) agent node containers (k3s)
`

// NewCmdClusterCreate returns a new cobra command
func NewCmdClusterCreate() *cobra.Command {

	createClusterOpts := &k3d.ClusterCreateOpts{}
	var updateDefaultKubeconfig, updateCurrentContext bool

	// create new command
	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a new cluster",
		Long:  clusterCreateDescription,
		Args:  cobra.RangeArgs(0, 1), // exactly one cluster name can be set (default: k3d.DefaultClusterName)
		Run: func(cmd *cobra.Command, args []string) {
			// parse args and flags
			cluster := parseCreateClusterCmd(cmd, args, createClusterOpts)

			// check if a cluster with that name exists already
			if _, err := k3dCluster.ClusterGet(cmd.Context(), runtimes.SelectedRuntime, cluster); err == nil {
				log.Fatalf("Failed to create cluster '%s' because a cluster with that name already exists", cluster.Name)
			}

			if !updateDefaultKubeconfig && updateCurrentContext {
				log.Infoln("--update-default-kubeconfig=false --> sets --switch-context=false")
				updateCurrentContext = false
			}

			// create cluster
			if updateDefaultKubeconfig {
				log.Debugln("'--update-default-kubeconfig set: enabling wait-for-server")
				cluster.CreateClusterOpts.WaitForServer = true
			}
			if err := k3dCluster.ClusterCreate(cmd.Context(), runtimes.SelectedRuntime, cluster); err != nil {
				// rollback if creation failed
				log.Errorln(err)
				log.Errorln("Failed to create cluster >>> Rolling Back")
				if err := k3dCluster.ClusterDelete(cmd.Context(), runtimes.SelectedRuntime, cluster); err != nil {
					log.Errorln(err)
					log.Fatalln("Cluster creation FAILED, also FAILED to rollback changes!")
				}
				log.Fatalln("Cluster creation FAILED, all changes have been rolled back!")
			}
			log.Infof("Cluster '%s' created successfully!", cluster.Name)

			if updateDefaultKubeconfig {
				log.Debugf("Updating default kubeconfig with a new context for cluster %s", cluster.Name)
				if _, err := k3dCluster.KubeconfigGetWrite(cmd.Context(), runtimes.SelectedRuntime, cluster, "", &k3dCluster.WriteKubeConfigOptions{UpdateExisting: true, OverwriteExisting: false, UpdateCurrentContext: updateCurrentContext}); err != nil {
					log.Warningln(err)
				}
			}

			// print information on how to use the cluster with kubectl
			log.Infoln("You can now use it like this:")
			if updateDefaultKubeconfig && !updateCurrentContext {
				fmt.Printf("kubectl config use-context %s\n", fmt.Sprintf("%s-%s", k3d.DefaultObjectNamePrefix, cluster.Name))
			} else if !updateCurrentContext {
				if runtime.GOOS == "windows" {
					fmt.Printf("$env:KUBECONFIG=(%s kubeconfig write %s)\n", os.Args[0], cluster.Name)
				} else {
					fmt.Printf("export KUBECONFIG=$(%s kubeconfig write %s)\n", os.Args[0], cluster.Name)
				}
			}
			fmt.Println("kubectl cluster-info")
		},
	}

	/*********
	 * Flags *
	 *********/
	cmd.Flags().StringP("config", "c", "", "Use a config file")
	if err := cmd.MarkFlagFilename("config"); err != nil {
		log.Fatalln("Failed to mark flag --config as filename")
	}
	cmd.Flags().String("api-port", "random", "Specify the Kubernetes API server port exposed on the LoadBalancer (Format: `[HOST:]HOSTPORT`)\n - Example: `k3d cluster create --servers 3 --api-port 0.0.0.0:6550`")
	cmd.Flags().IntP("servers", "s", 1, "Specify how many servers you want to create")
	cmd.Flags().IntP("agents", "a", 0, "Specify how many agents you want to create")
	cmd.Flags().StringP("image", "i", fmt.Sprintf("%s:%s", k3d.DefaultK3sImageRepo, version.GetK3sVersion(false)), "Specify k3s image that you want to use for the nodes")
	cmd.Flags().String("network", "", "Join an existing network")
	cmd.Flags().String("token", "", "Specify a cluster token. By default, we generate one.")
	cmd.Flags().StringArrayP("volume", "v", nil, "Mount volumes into the nodes (Format: `[SOURCE:]DEST[@NODEFILTER[;NODEFILTER...]]`\n - Example: `k3d cluster create --agents 2 -v /my/path@agent[0,1] -v /tmp/test:/tmp/other@server[0]`")
	cmd.Flags().StringArrayP("port", "p", nil, "Map ports from the node containers to the host (Format: `[HOST:][HOSTPORT:]CONTAINERPORT[/PROTOCOL][@NODEFILTER]`)\n - Example: `k3d cluster create --agents 2 -p 8080:80@agent[0] -p 8081@agent[1]`")
	cmd.Flags().BoolVar(&createClusterOpts.WaitForServer, "wait", true, "Wait for the server(s) to be ready before returning. Use '--timeout DURATION' to not wait forever.")
	cmd.Flags().DurationVar(&createClusterOpts.Timeout, "timeout", 0*time.Second, "Rollback changes if cluster couldn't be created in specified duration.")
	cmd.Flags().BoolVar(&updateDefaultKubeconfig, "update-default-kubeconfig", true, "Directly update the default kubeconfig with the new cluster's context")
	cmd.Flags().BoolVar(&updateCurrentContext, "switch-context", true, "Directly switch the default kubeconfig's current-context to the new cluster's context (requires --update-default-kubeconfig)")
	cmd.Flags().BoolVar(&createClusterOpts.DisableLoadBalancer, "no-lb", false, "Disable the creation of a LoadBalancer in front of the server nodes")

	/* Image Importing */
	cmd.Flags().BoolVar(&createClusterOpts.DisableImageVolume, "no-image-volume", false, "Disable the creation of a volume for importing images")

	/* Multi Server Configuration */

	// multi-server - datastore
	// TODO: implement multi-server setups with external data store
	// cmd.Flags().String("datastore-endpoint", "", "[WIP] Specify external datastore endpoint (e.g. for multi server clusters)")
	/*
		cmd.Flags().String("datastore-network", "", "Specify container network where we can find the datastore-endpoint (add a connection)")

		// TODO: set default paths and hint, that one should simply mount the files using --volume flag
		cmd.Flags().String("datastore-cafile", "", "Specify external datastore's TLS Certificate Authority (CA) file")
		cmd.Flags().String("datastore-certfile", "", "Specify external datastore's TLS certificate file'")
		cmd.Flags().String("datastore-keyfile", "", "Specify external datastore's TLS key file'")
	*/

	/* k3s */
	cmd.Flags().StringArrayVar(&createClusterOpts.K3sServerArgs, "k3s-server-arg", nil, "Additional args passed to the `k3s server` command on server nodes (new flag per arg)")
	cmd.Flags().StringArrayVar(&createClusterOpts.K3sAgentArgs, "k3s-agent-arg", nil, "Additional args passed to the `k3s agent` command on agent nodes (new flag per arg)")

	/* Subcommands */

	// done
	return cmd
}

// parseCreateClusterCmd parses the command input into variables required to create a cluster
func parseCreateClusterCmd(cmd *cobra.Command, args []string, createClusterOpts *k3d.ClusterCreateOpts) *k3d.Cluster {

	/********************************
	 * Parse and validate arguments *
	 ********************************/

	clustername := k3d.DefaultClusterName
	if len(args) != 0 {
		clustername = args[0]
	}
	if err := cluster.CheckName(clustername); err != nil {
		log.Fatal(err)
	}

	/****************************
	 * Parse and validate flags *
	 ****************************/

	// --image
	image, err := cmd.Flags().GetString("image")
	if err != nil {
		log.Errorln("No image specified")
		log.Fatalln(err)
	}
	if image == "latest" {
		image = version.GetK3sVersion(true)
	}

	// --servers
	serverCount, err := cmd.Flags().GetInt("servers")
	if err != nil {
		log.Fatalln(err)
	}

	// --agents
	agentCount, err := cmd.Flags().GetInt("agents")
	if err != nil {
		log.Fatalln(err)
	}

	// --network
	networkName, err := cmd.Flags().GetString("network")
	if err != nil {
		log.Fatalln(err)
	}
	network := k3d.ClusterNetwork{}
	if networkName != "" {
		network.Name = networkName
		network.External = true
	}
	if networkName == "host" && (serverCount+agentCount) > 1 {
		log.Fatalln("Can only run a single node in hostnetwork mode")
	}

	// --token
	token, err := cmd.Flags().GetString("token")
	if err != nil {
		log.Fatalln(err)
	}

	// --timeout
	if cmd.Flags().Changed("timeout") && createClusterOpts.Timeout <= 0*time.Second {
		log.Fatalln("--timeout DURATION must be >= 1s")
	}

	// --api-port
	apiPort, err := cmd.Flags().GetString("api-port")
	if err != nil {
		log.Fatalln(err)
	}

	// parse the port mapping
	exposeAPI, err := cliutil.ParseAPIPort(apiPort)
	if err != nil {
		log.Fatalln(err)
	}
	if exposeAPI.Host == "" {
		exposeAPI.Host = k3d.DefaultAPIHost
	}
	if exposeAPI.HostIP == "" {
		exposeAPI.HostIP = k3d.DefaultAPIHost
	}
	if networkName == "host" {
		// in hostNetwork mode, we're not going to map a hostport. Here it should always use 6443.
		// Note that hostNetwork mode is super inflexible and since we don't change the backend port (on the container), it will only be one hostmode cluster allowed.
		exposeAPI.Port = k3d.DefaultAPIPort
	}

	// --volume
	volumeFlags, err := cmd.Flags().GetStringArray("volume")
	if err != nil {
		log.Fatalln(err)
	}

	// volumeFilterMap will map volume mounts to applied node filters
	volumeFilterMap := make(map[string][]string, 1)
	for _, volumeFlag := range volumeFlags {

		// split node filter from the specified volume
		volume, filters, err := cliutil.SplitFiltersFromFlag(volumeFlag)
		if err != nil {
			log.Fatalln(err)
		}

		// validate the specified volume mount and return it in SRC:DEST format
		volume, err = cliutil.ValidateVolumeMount(runtimes.SelectedRuntime, volume)
		if err != nil {
			log.Fatalln(err)
		}

		// create new entry or append filter to existing entry
		if _, exists := volumeFilterMap[volume]; exists {
			volumeFilterMap[volume] = append(volumeFilterMap[volume], filters...)
		} else {
			volumeFilterMap[volume] = filters
		}
	}

	// --port
	portFlags, err := cmd.Flags().GetStringArray("port")
	if err != nil {
		log.Fatalln(err)
	}
	portFilterMap := make(map[string][]string, 1)
	for _, portFlag := range portFlags {
		// split node filter from the specified volume
		portmap, filters, err := cliutil.SplitFiltersFromFlag(portFlag)
		if err != nil {
			log.Fatalln(err)
		}

		if len(filters) > 1 {
			log.Fatalln("Can only apply a Portmap to one node")
		}

		// the same portmapping can't be applied to multiple nodes

		// validate the specified volume mount and return it in SRC:DEST format
		portmap, err = cliutil.ValidatePortMap(portmap)
		if err != nil {
			log.Fatalln(err)
		}

		// create new entry or append filter to existing entry
		if _, exists := portFilterMap[portmap]; exists {
			log.Fatalln("Same Portmapping can not be used for multiple nodes")
		} else {
			portFilterMap[portmap] = filters
		}
	}

	log.Debugf("PortFilterMap: %+v", portFilterMap)

	/********************
	 *									*
	 * generate cluster *
	 *									*
	 ********************/

	cluster := &k3d.Cluster{
		Name:              clustername,
		Network:           network,
		Token:             token,
		CreateClusterOpts: createClusterOpts,
		ExposeAPI:         exposeAPI,
	}

	// generate list of nodes
	cluster.Nodes = []*k3d.Node{}

	// ServerLoadBalancer
	if !createClusterOpts.DisableLoadBalancer {
		cluster.ServerLoadBalancer = &k3d.Node{
			Role: k3d.LoadBalancerRole,
		}
	}

	/****************
	 * Server Nodes *
	 ****************/

	for i := 0; i < serverCount; i++ {
		node := k3d.Node{
			Role:       k3d.ServerRole,
			Image:      image,
			Args:       createClusterOpts.K3sServerArgs,
			ServerOpts: k3d.ServerOpts{},
		}

		// TODO: by default, we don't expose an API port: should we change that?
		// -> if we want to change that, simply add the exposeAPI struct here

		// first server node will be init node if we have more than one server specified but no external datastore
		if i == 0 && serverCount > 1 {
			node.ServerOpts.IsInit = true
			cluster.InitNode = &node
		}

		// append node to list
		cluster.Nodes = append(cluster.Nodes, &node)
	}

	/****************
	 * Agent Nodes *
	 ****************/

	for i := 0; i < agentCount; i++ {
		node := k3d.Node{
			Role:  k3d.AgentRole,
			Image: image,
			Args:  createClusterOpts.K3sAgentArgs,
		}

		cluster.Nodes = append(cluster.Nodes, &node)
	}

	// append volumes
	for volume, filters := range volumeFilterMap {
		nodes, err := cliutil.FilterNodes(cluster.Nodes, filters)
		if err != nil {
			log.Fatalln(err)
		}
		for _, node := range nodes {
			node.Volumes = append(node.Volumes, volume)
		}
	}

	// append ports
	nodeCount := serverCount + agentCount
	nodeList := cluster.Nodes
	if !createClusterOpts.DisableLoadBalancer {
		nodeCount++
		nodeList = append(nodeList, cluster.ServerLoadBalancer)
	}
	for portmap, filters := range portFilterMap {
		if len(filters) == 0 && (nodeCount) > 1 {
			log.Fatalf("Malformed portmapping '%s' lacks a node filter, but there is more than one node (including the loadbalancer, if there is any).", portmap)
		}
		nodes, err := cliutil.FilterNodes(nodeList, filters)
		if err != nil {
			log.Fatalln(err)
		}
		for _, node := range nodes {
			node.Ports = append(node.Ports, portmap)
		}
	}

	/**********************
	 * Utility Containers *
	 **********************/
	// ...

	return cluster
}
