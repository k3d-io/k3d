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
	k3dCluster "github.com/rancher/k3d/v3/pkg/cluster"
	"github.com/rancher/k3d/v3/pkg/config"
	conf "github.com/rancher/k3d/v3/pkg/config/v1alpha1"
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

	simpleConfig := &conf.SimpleConfig{}
	var configFile string

	// create new command
	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a new cluster",
		Long:  clusterCreateDescription,
		Args:  cobra.RangeArgs(0, 1), // exactly one cluster name can be set (default: k3d.DefaultClusterName)
		Run: func(cmd *cobra.Command, args []string) {

			// parse args and flags
			simpleConfig = parseCreateClusterCmd(cmd, args, simpleConfig)

			log.Debugf("========== Simple Config ==========\n%+v\n==========================\n", simpleConfig)

			if configFile != "" {
				configFromFile, err := config.ReadConfig(configFile)
				if err != nil {
					log.Fatalln(err)
				}
				simpleConfig, err = config.MergeSimple(*simpleConfig, configFromFile.(conf.SimpleConfig))
				if err != nil {
					log.Fatalln(err)
				}
			}

			log.Debugf("========== Merged Simple Config ==========\n%+v\n==========================\n", simpleConfig)
			clusterConfig, err := config.TransformSimpleToClusterConfig(cmd.Context(), runtimes.SelectedRuntime, *simpleConfig)
			if err != nil {
				log.Fatalln(err)
			}
			log.Debugf("===== Cluster Config =====\n%+v\n===== ===== =====\n", clusterConfig)
			if err := config.ValidateClusterConfig(cmd.Context(), runtimes.SelectedRuntime, *clusterConfig); err != nil {
				log.Fatalln("Failed Cluster Configuration Validation: ", err)
			}

			// check if a cluster with that name exists already
			if _, err := k3dCluster.ClusterGet(cmd.Context(), runtimes.SelectedRuntime, &clusterConfig.Cluster); err == nil {
				log.Fatalf("Failed to create cluster '%s' because a cluster with that name already exists", clusterConfig.Cluster.Name)
			}

			if !simpleConfig.Options.KubeconfigOptions.UpdateDefaultKubeconfig && simpleConfig.Options.KubeconfigOptions.SwitchCurrentContext {
				log.Infoln("--update-default-kubeconfig=false --> sets --switch-context=false")
				simpleConfig.Options.KubeconfigOptions.SwitchCurrentContext = false
			}

			// create cluster
			if simpleConfig.Options.KubeconfigOptions.UpdateDefaultKubeconfig {
				log.Debugln("'--update-default-kubeconfig set: enabling wait-for-server")
				clusterConfig.Cluster.ClusterCreateOpts.WaitForServer = true
			}
			if err := k3dCluster.ClusterCreate(cmd.Context(), runtimes.SelectedRuntime, &clusterConfig.Cluster); err != nil {
				// rollback if creation failed
				log.Errorln(err)
				if simpleConfig.Options.K3dOptions.NoRollback {
					log.Fatalln("Cluster creation FAILED, rollback deactivated.")
				}
				// rollback if creation failed
				log.Errorln("Failed to create cluster >>> Rolling Back")
				if err := k3dCluster.ClusterDelete(cmd.Context(), runtimes.SelectedRuntime, &clusterConfig.Cluster); err != nil {
					log.Errorln(err)
					log.Fatalln("Cluster creation FAILED, also FAILED to rollback changes!")
				}
				log.Fatalln("Cluster creation FAILED, all changes have been rolled back!")
			}
			log.Infof("Cluster '%s' created successfully!", clusterConfig.Cluster.Name)

			if simpleConfig.Options.KubeconfigOptions.UpdateDefaultKubeconfig {
				log.Debugf("Updating default kubeconfig with a new context for cluster %s", clusterConfig.Cluster.Name)
				if _, err := k3dCluster.KubeconfigGetWrite(cmd.Context(), runtimes.SelectedRuntime, &clusterConfig.Cluster, "", &k3dCluster.WriteKubeConfigOptions{UpdateExisting: true, OverwriteExisting: false, UpdateCurrentContext: simpleConfig.Options.KubeconfigOptions.SwitchCurrentContext}); err != nil {
					log.Warningln(err)
				}
			}

			// print information on how to use the cluster with kubectl
			log.Infoln("You can now use it like this:")
			if simpleConfig.Options.KubeconfigOptions.UpdateDefaultKubeconfig && !simpleConfig.Options.KubeconfigOptions.SwitchCurrentContext {
				fmt.Printf("kubectl config use-context %s\n", fmt.Sprintf("%s-%s", k3d.DefaultObjectNamePrefix, clusterConfig.Cluster.Name))
			} else if !simpleConfig.Options.KubeconfigOptions.SwitchCurrentContext {
				if runtime.GOOS == "windows" {
					fmt.Printf("$env:KUBECONFIG=(%s kubeconfig write %s)\n", os.Args[0], clusterConfig.Cluster.Name)
				} else {
					fmt.Printf("export KUBECONFIG=$(%s kubeconfig write %s)\n", os.Args[0], clusterConfig.Cluster.Name)
				}
			}
			fmt.Println("kubectl cluster-info")
		},
	}

	/*********
	 * Flags *
	 *********/
	cmd.Flags().String("api-port", "random", "Specify the Kubernetes API server port exposed on the LoadBalancer (Format: `[HOST:]HOSTPORT`)\n - Example: `k3d cluster create --servers 3 --api-port 0.0.0.0:6550`")
	cmd.Flags().IntVarP(&simpleConfig.Servers, "servers", "s", 1, "Specify how many servers you want to create")
	cmd.Flags().IntVarP(&simpleConfig.Agents, "agents", "a", 0, "Specify how many agents you want to create")
	cmd.Flags().StringVarP(&simpleConfig.Image, "image", "i", fmt.Sprintf("%s:%s", k3d.DefaultK3sImageRepo, version.GetK3sVersion(false)), "Specify k3s image that you want to use for the nodes")
	cmd.Flags().StringVar(&simpleConfig.Network, "network", "", "Join an existing network")
	cmd.Flags().StringVar(&simpleConfig.ClusterToken, "token", "", "Specify a cluster token. By default, we generate one.")
	cmd.Flags().StringArrayP("volume", "v", nil, "Mount volumes into the nodes (Format: `[SOURCE:]DEST[@NODEFILTER[;NODEFILTER...]]`\n - Example: `k3d cluster create --agents 2 -v /my/path@agent[0,1] -v /tmp/test:/tmp/other@server[0]`")
	cmd.Flags().StringArrayP("port", "p", nil, "Map ports from the node containers to the host (Format: `[HOST:][HOSTPORT:]CONTAINERPORT[/PROTOCOL][@NODEFILTER]`)\n - Example: `k3d cluster create --agents 2 -p 8080:80@agent[0] -p 8081@agent[1]`")
	cmd.Flags().StringArrayP("label", "l", nil, "Add label to node container (Format: `KEY[=VALUE][@NODEFILTER[;NODEFILTER...]]`\n - Example: `k3d cluster create --agents 2 -l \"my.label@agent[0,1]\" -v \"other.label=somevalue@server[0]\"`")
	cmd.Flags().BoolVar(&simpleConfig.Options.K3dOptions.Wait, "wait", true, "Wait for the server(s) to be ready before returning. Use '--timeout DURATION' to not wait forever.")
	cmd.Flags().DurationVar(&simpleConfig.Options.K3dOptions.Timeout, "timeout", 0*time.Second, "Rollback changes if cluster couldn't be created in specified duration.")
	cmd.Flags().BoolVar(&simpleConfig.Options.KubeconfigOptions.UpdateDefaultKubeconfig, "update-default-kubeconfig", true, "Directly update the default kubeconfig with the new cluster's context")
	cmd.Flags().BoolVar(&simpleConfig.Options.KubeconfigOptions.SwitchCurrentContext, "switch-context", true, "Directly switch the default kubeconfig's current-context to the new cluster's context (requires --update-default-kubeconfig)")
	cmd.Flags().BoolVar(&simpleConfig.Options.K3dOptions.DisableLoadbalancer, "no-lb", false, "Disable the creation of a LoadBalancer in front of the server nodes")
	cmd.Flags().BoolVar(&simpleConfig.Options.K3dOptions.NoRollback, "no-rollback", false, "Disable the automatic rollback actions, if anything goes wrong")
	cmd.Flags().BoolVar(&simpleConfig.Options.K3dOptions.PrepDisableHostIPInjection, "no-hostip", false, "Disable the automatic injection of the Host IP as 'host.k3d.internal' into the containers and CoreDNS")

	/* Image Importing */
	cmd.Flags().BoolVar(&simpleConfig.Options.K3dOptions.DisableImageVolume, "no-image-volume", false, "Disable the creation of a volume for importing images")

	/* Config File */
	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Path of a config file to use")
	if err := cobra.MarkFlagFilename(cmd.Flags(), "config", "yaml", "yml"); err != nil {
		log.Fatalln("Failed to mark flag 'config' as filename flag")
	}

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
	cmd.Flags().StringArrayVar(&simpleConfig.Options.K3sOptions.ExtraServerArgs, "k3s-server-arg", nil, "Additional args passed to the `k3s server` command on server nodes (new flag per arg)")
	cmd.Flags().StringArrayVar(&simpleConfig.Options.K3sOptions.ExtraAgentArgs, "k3s-agent-arg", nil, "Additional args passed to the `k3s agent` command on agent nodes (new flag per arg)")

	/* Subcommands */

	// done
	return cmd
}

// parseCreateClusterCmd parses the command input into variables required to create a cluster
func parseCreateClusterCmd(cmd *cobra.Command, args []string, simpleConfig *conf.SimpleConfig) *conf.SimpleConfig {

	/********************************
	 * Parse and validate arguments *
	 ********************************/

	clustername := k3d.DefaultClusterName
	if len(args) != 0 {
		clustername = args[0]
	}

	simpleConfig.Name = clustername

	/****************************
	 * Parse and validate flags *
	 ****************************/

	// -> IMAGE
	if simpleConfig.Image == "latest" {
		simpleConfig.Image = version.GetK3sVersion(true)
	}

	// -> WAIT TIMEOUT
	if cmd.Flags().Changed("timeout") && simpleConfig.Options.K3dOptions.Timeout <= 0*time.Second {
		log.Fatalln("--timeout DURATION must be >= 1s")
	}

	// -> API-PORT
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

	simpleConfig.ExposeAPI = exposeAPI

	// -> VOLUMES
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

		// create new entry or append filter to existing entry
		if _, exists := volumeFilterMap[volume]; exists {
			volumeFilterMap[volume] = append(volumeFilterMap[volume], filters...)
		} else {
			volumeFilterMap[volume] = filters
		}
	}

	for volume, nodeFilters := range volumeFilterMap {
		simpleConfig.Volumes = append(simpleConfig.Volumes, conf.VolumeWithNodeFilters{
			Volume:      volume,
			NodeFilters: nodeFilters,
		})
		log.Debugf("%+v, %+v", nodeFilters, volume)
	}

	// -> PORTS
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

		// create new entry or append filter to existing entry
		if _, exists := portFilterMap[portmap]; exists {
			log.Fatalln("Same Portmapping can not be used for multiple nodes")
		} else {
			portFilterMap[portmap] = filters
		}
	}

	for port, nodeFilters := range portFilterMap {
		simpleConfig.Ports = append(simpleConfig.Ports, conf.PortWithNodeFilters{
			Port:        port,
			NodeFilters: nodeFilters,
		})
		log.Debugf("Port: %s, Filters: %+v", port, nodeFilters)
	}

	log.Debugf("PortFilterMap: %+v", portFilterMap)

	// --label
	labelFlags, err := cmd.Flags().GetStringArray("label")
	if err != nil {
		log.Fatalln(err)
	}

	// labelFilterMap will add container label to applied node filters
	labelFilterMap := make(map[string][]string, 1)
	for _, labelFlag := range labelFlags {

		// split node filter from the specified label
		label, nodeFilters, err := cliutil.SplitFiltersFromFlag(labelFlag)
		if err != nil {
			log.Fatalln(err)
		}

		// create new entry or append filter to existing entry
		if _, exists := labelFilterMap[label]; exists {
			labelFilterMap[label] = append(labelFilterMap[label], nodeFilters...)
		} else {
			labelFilterMap[label] = nodeFilters
		}
	}

	for label, nodeFilters := range labelFilterMap {
		simpleConfig.Labels = append(simpleConfig.Labels, conf.LabelWithNodeFilters{
			Label:       label,
			NodeFilters: nodeFilters,
		})
		log.Tracef("Label: %s, Filters: %+v", label, nodeFilters)
	}

	log.Tracef("LabelFilterMap: %+v", labelFilterMap)

	return simpleConfig
}
