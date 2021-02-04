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
	"strings"
	"time"

	"github.com/spf13/cobra"

	cliutil "github.com/rancher/k3d/v4/cmd/util"
	k3dCluster "github.com/rancher/k3d/v4/pkg/client"
	"github.com/rancher/k3d/v4/pkg/config"
	conf "github.com/rancher/k3d/v4/pkg/config/v1alpha1"
	"github.com/rancher/k3d/v4/pkg/runtimes"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	"github.com/rancher/k3d/v4/version"

	log "github.com/sirupsen/logrus"
)

const clusterCreateDescription = `
Create a new k3s cluster with containerized nodes (k3s in docker).
Every cluster will consist of one or more containers:
	- 1 (or more) server node container (k3s)
	- (optionally) 1 loadbalancer container as the entrypoint to the cluster (nginx)
	- (optionally) 1 (or more) agent node containers (k3s)
`

// flags that go through some pre-processing before transforming them to config
type preProcessedFlags struct {
	APIPort     string
	Volumes     []string
	Ports       []string
	Labels      []string
	Env         []string
	RegistryUse []string
}

// NewCmdClusterCreate returns a new cobra command
func NewCmdClusterCreate() *cobra.Command {

	cliConfig := &conf.SimpleConfig{}
	var configFile string
	ppFlags := &preProcessedFlags{}

	// create new command
	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a new cluster",
		Long:  clusterCreateDescription,
		Args:  cobra.RangeArgs(0, 1), // exactly one cluster name can be set (default: k3d.DefaultClusterName)
		Run: func(cmd *cobra.Command, args []string) {

			/*********************
			 * CLI Configuration *
			 *********************/
			parseCreateClusterCmd(cmd, args, cliConfig, ppFlags)

			/************************
			 * Merge Configurations *
			 ************************/
			log.Debugf("========== Simple Config ==========\n%+v\n==========================\n", cliConfig)

			if configFile != "" {
				configFromFile, err := config.ReadConfig(configFile)
				if err != nil {
					log.Fatalln(err)
				}
				cliConfig, err = config.MergeSimple(*cliConfig, configFromFile.(conf.SimpleConfig))
				if err != nil {
					log.Fatalln(err)
				}
			}

			log.Debugf("========== Merged Simple Config ==========\n%+v\n==========================\n", cliConfig)

			/***********************************************
			 * Transform, Process & Validate Configuration *
			 ***********************************************/
			clusterConfig, err := config.TransformSimpleToClusterConfig(cmd.Context(), runtimes.SelectedRuntime, *cliConfig)
			if err != nil {
				log.Fatalln(err)
			}
			log.Debugf("===== Merged Cluster Config =====\n%+v\n===== ===== =====\n", clusterConfig)

			clusterConfig, err = config.ProcessClusterConfig(*clusterConfig)
			if err != nil {
				log.Fatalln(err)
			}
			log.Debugf("===== Processed Cluster Config =====\n%+v\n===== ===== =====\n", clusterConfig)

			if err := config.ValidateClusterConfig(cmd.Context(), runtimes.SelectedRuntime, *clusterConfig); err != nil {
				log.Fatalln("Failed Cluster Configuration Validation: ", err)
			}

			/**************************************
			 * Create cluster if it doesn't exist *
			 **************************************/

			// check if a cluster with that name exists already
			if _, err := k3dCluster.ClusterGet(cmd.Context(), runtimes.SelectedRuntime, &clusterConfig.Cluster); err == nil {
				log.Fatalf("Failed to create cluster '%s' because a cluster with that name already exists", clusterConfig.Cluster.Name)
			}

			// create cluster
			if clusterConfig.KubeconfigOpts.UpdateDefaultKubeconfig {
				log.Debugln("'--kubeconfig-update-default set: enabling wait-for-server")
				clusterConfig.ClusterCreateOpts.WaitForServer = true
			}
			//if err := k3dCluster.ClusterCreate(cmd.Context(), runtimes.SelectedRuntime, &clusterConfig.Cluster, &clusterConfig.ClusterCreateOpts); err != nil {
			if err := k3dCluster.ClusterRun(cmd.Context(), runtimes.SelectedRuntime, clusterConfig); err != nil {
				// rollback if creation failed
				log.Errorln(err)
				if cliConfig.Options.K3dOptions.NoRollback { // TODO: move rollback mechanics to pkg/
					log.Fatalln("Cluster creation FAILED, rollback deactivated.")
				}
				// rollback if creation failed
				log.Errorln("Failed to create cluster >>> Rolling Back")
				if err := k3dCluster.ClusterDelete(cmd.Context(), runtimes.SelectedRuntime, &clusterConfig.Cluster, k3d.ClusterDeleteOpts{SkipRegistryCheck: true}); err != nil {
					log.Errorln(err)
					log.Fatalln("Cluster creation FAILED, also FAILED to rollback changes!")
				}
				log.Fatalln("Cluster creation FAILED, all changes have been rolled back!")
			}
			log.Infof("Cluster '%s' created successfully!", clusterConfig.Cluster.Name)

			/**************
			 * Kubeconfig *
			 **************/

			if clusterConfig.KubeconfigOpts.UpdateDefaultKubeconfig && clusterConfig.KubeconfigOpts.SwitchCurrentContext {
				log.Infoln("--kubeconfig-update-default=false --> sets --kubeconfig-switch-context=false")
				clusterConfig.KubeconfigOpts.SwitchCurrentContext = false
			}

			if clusterConfig.KubeconfigOpts.UpdateDefaultKubeconfig {
				log.Debugf("Updating default kubeconfig with a new context for cluster %s", clusterConfig.Cluster.Name)
				if _, err := k3dCluster.KubeconfigGetWrite(cmd.Context(), runtimes.SelectedRuntime, &clusterConfig.Cluster, "", &k3dCluster.WriteKubeConfigOptions{UpdateExisting: true, OverwriteExisting: false, UpdateCurrentContext: cliConfig.Options.KubeconfigOptions.SwitchCurrentContext}); err != nil {
					log.Warningln(err)
				}
			}

			/*****************
			 * User Feedback *
			 *****************/

			// print information on how to use the cluster with kubectl
			log.Infoln("You can now use it like this:")
			if clusterConfig.KubeconfigOpts.UpdateDefaultKubeconfig && !clusterConfig.KubeconfigOpts.SwitchCurrentContext {
				fmt.Printf("kubectl config use-context %s\n", fmt.Sprintf("%s-%s", k3d.DefaultObjectNamePrefix, clusterConfig.Cluster.Name))
			} else if !clusterConfig.KubeconfigOpts.SwitchCurrentContext {
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
	cmd.Flags().StringVar(&ppFlags.APIPort, "api-port", "random", "Specify the Kubernetes API server port exposed on the LoadBalancer (Format: `[HOST:]HOSTPORT`)\n - Example: `k3d cluster create --servers 3 --api-port 0.0.0.0:6550`")
	cmd.Flags().IntVarP(&cliConfig.Servers, "servers", "s", 1, "Specify how many servers you want to create")
	cmd.Flags().IntVarP(&cliConfig.Agents, "agents", "a", 0, "Specify how many agents you want to create")
	cmd.Flags().StringVarP(&cliConfig.Image, "image", "i", fmt.Sprintf("%s:%s", k3d.DefaultK3sImageRepo, version.GetK3sVersion(false)), "Specify k3s image that you want to use for the nodes")
	cmd.Flags().StringVar(&cliConfig.Network, "network", "", "Join an existing network")
	cmd.Flags().StringVar(&cliConfig.ClusterToken, "token", "", "Specify a cluster token. By default, we generate one.")
	cmd.Flags().StringArrayVarP(&ppFlags.Volumes, "volume", "v", nil, "Mount volumes into the nodes (Format: `[SOURCE:]DEST[@NODEFILTER[;NODEFILTER...]]`\n - Example: `k3d cluster create --agents 2 -v /my/path@agent[0,1] -v /tmp/test:/tmp/other@server[0]`")
	cmd.Flags().StringArrayVarP(&ppFlags.Ports, "port", "p", nil, "Map ports from the node containers to the host (Format: `[HOST:][HOSTPORT:]CONTAINERPORT[/PROTOCOL][@NODEFILTER]`)\n - Example: `k3d cluster create --agents 2 -p 8080:80@agent[0] -p 8081@agent[1]`")
	cmd.Flags().StringArrayVarP(&ppFlags.Labels, "label", "l", nil, "Add label to node container (Format: `KEY[=VALUE][@NODEFILTER[;NODEFILTER...]]`\n - Example: `k3d cluster create --agents 2 -l \"my.label@agent[0,1]\" -v \"other.label=somevalue@server[0]\"`")
	cmd.Flags().BoolVar(&cliConfig.Options.K3dOptions.Wait, "wait", true, "Wait for the server(s) to be ready before returning. Use '--timeout DURATION' to not wait forever.")
	cmd.Flags().DurationVar(&cliConfig.Options.K3dOptions.Timeout, "timeout", 0*time.Second, "Rollback changes if cluster couldn't be created in specified duration.")
	cmd.Flags().BoolVar(&cliConfig.Options.KubeconfigOptions.UpdateDefaultKubeconfig, "kubeconfig-update-default", true, "Directly update the default kubeconfig with the new cluster's context")
	cmd.Flags().BoolVar(&cliConfig.Options.KubeconfigOptions.SwitchCurrentContext, "kubeconfig-switch-context", true, "Directly switch the default kubeconfig's current-context to the new cluster's context (requires --kubeconfig-update-default)")
	cmd.Flags().BoolVar(&cliConfig.Options.K3dOptions.DisableLoadbalancer, "no-lb", false, "Disable the creation of a LoadBalancer in front of the server nodes")
	cmd.Flags().BoolVar(&cliConfig.Options.K3dOptions.NoRollback, "no-rollback", false, "Disable the automatic rollback actions, if anything goes wrong")
	cmd.Flags().BoolVar(&cliConfig.Options.K3dOptions.PrepDisableHostIPInjection, "no-hostip", false, "Disable the automatic injection of the Host IP as 'host.k3d.internal' into the containers and CoreDNS")
	cmd.Flags().StringVar(&cliConfig.Options.Runtime.GPURequest, "gpus", "", "GPU devices to add to the cluster node containers ('all' to pass all GPUs) [From docker]")
	cmd.Flags().StringArrayVarP(&ppFlags.Env, "env", "e", nil, "Add environment variables to nodes (Format: `KEY[=VALUE][@NODEFILTER[;NODEFILTER...]]`\n - Example: `k3d cluster create --agents 2 -e \"HTTP_PROXY=my.proxy.com\" -e \"SOME_KEY=SOME_VAL@server[0]\"`")

	/* Image Importing */
	cmd.Flags().BoolVar(&cliConfig.Options.K3dOptions.DisableImageVolume, "no-image-volume", false, "Disable the creation of a volume for importing images")

	/* Config File */
	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Path of a config file to use")
	if err := cobra.MarkFlagFilename(cmd.Flags(), "config", "yaml", "yml"); err != nil {
		log.Fatalln("Failed to mark flag 'config' as filename flag")
	}

	/* Registry */
	cmd.Flags().StringArrayVar(&cliConfig.Registries.Use, "registry-use", nil, "Connect to one or more k3d-managed registries running locally")
	cmd.Flags().BoolVar(&cliConfig.Registries.Create, "registry-create", false, "Create a k3d-managed registry and connect it to the cluster")
	cmd.Flags().StringVar(&cliConfig.Registries.Config, "registry-config", "", "Specify path to an extra registries.yaml file")

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
	cmd.Flags().StringArrayVar(&cliConfig.Options.K3sOptions.ExtraServerArgs, "k3s-server-arg", nil, "Additional args passed to the `k3s server` command on server nodes (new flag per arg)")
	cmd.Flags().StringArrayVar(&cliConfig.Options.K3sOptions.ExtraAgentArgs, "k3s-agent-arg", nil, "Additional args passed to the `k3s agent` command on agent nodes (new flag per arg)")

	/* Subcommands */

	// done
	return cmd
}

// parseCreateClusterCmd parses the command input into variables required to create a cluster
func parseCreateClusterCmd(cmd *cobra.Command, args []string, cliConfig *conf.SimpleConfig, ppFlags *preProcessedFlags) {

	/********************************
	 * Parse and validate arguments *
	 ********************************/

	if len(args) != 0 {
		cliConfig.Name = args[0]
	}

	/****************************
	 * Parse and validate flags *
	 ****************************/

	// -> WAIT TIMEOUT // TODO: timeout to be validated in pkg/
	if cmd.Flags().Changed("timeout") && cliConfig.Options.K3dOptions.Timeout <= 0*time.Second {
		log.Fatalln("--timeout DURATION must be >= 1s")
	}

	// -> API-PORT
	// parse the port mapping
	exposeAPI, err := cliutil.ParsePortExposureSpec(ppFlags.APIPort, k3d.DefaultAPIPort)
	if err != nil {
		log.Fatalln(err)
	}
	cliConfig.ExposeAPI = conf.SimpleExposureOpts{
		Host:     exposeAPI.Host,
		HostIP:   exposeAPI.Binding.HostIP,
		HostPort: exposeAPI.Binding.HostPort,
	}

	// -> VOLUMES
	// volumeFilterMap will map volume mounts to applied node filters
	volumeFilterMap := make(map[string][]string, 1)
	for _, volumeFlag := range ppFlags.Volumes {

		// split node filter from the specified volume
		volume, filters, err := cliutil.SplitFiltersFromFlag(volumeFlag)
		if err != nil {
			log.Fatalln(err)
		}

		if strings.Contains(volume, k3d.DefaultRegistriesFilePath) && (cliConfig.Registries.Create || cliConfig.Registries.Config != "" || len(cliConfig.Registries.Use) != 0) {
			log.Warnf("Seems like you're mounting a file at '%s' while also using a referenced registries config or k3d-managed registries: Your mounted file will probably be overwritten!", k3d.DefaultRegistriesFilePath)
		}

		// create new entry or append filter to existing entry
		if _, exists := volumeFilterMap[volume]; exists {
			volumeFilterMap[volume] = append(volumeFilterMap[volume], filters...)
		} else {
			volumeFilterMap[volume] = filters
		}
	}

	for volume, nodeFilters := range volumeFilterMap {
		cliConfig.Volumes = append(cliConfig.Volumes, conf.VolumeWithNodeFilters{
			Volume:      volume,
			NodeFilters: nodeFilters,
		})
	}

	log.Tracef("VolumeFilterMap: %+v", volumeFilterMap)

	// -> PORTS
	portFilterMap := make(map[string][]string, 1)
	for _, portFlag := range ppFlags.Ports {
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
		cliConfig.Ports = append(cliConfig.Ports, conf.PortWithNodeFilters{
			Port:        port,
			NodeFilters: nodeFilters,
		})
	}

	log.Tracef("PortFilterMap: %+v", portFilterMap)

	// --label
	// labelFilterMap will add container label to applied node filters
	labelFilterMap := make(map[string][]string, 1)
	for _, labelFlag := range ppFlags.Labels {

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
		cliConfig.Labels = append(cliConfig.Labels, conf.LabelWithNodeFilters{
			Label:       label,
			NodeFilters: nodeFilters,
		})
	}

	log.Tracef("LabelFilterMap: %+v", labelFilterMap)

	// --env
	// envFilterMap will add container env vars to applied node filters
	envFilterMap := make(map[string][]string, 1)
	for _, envFlag := range ppFlags.Env {

		// split node filter from the specified env var
		env, filters, err := cliutil.SplitFiltersFromFlag(envFlag)
		if err != nil {
			log.Fatalln(err)
		}

		// create new entry or append filter to existing entry
		if _, exists := envFilterMap[env]; exists {
			envFilterMap[env] = append(envFilterMap[env], filters...)
		} else {
			envFilterMap[env] = filters
		}
	}

	for envVar, nodeFilters := range envFilterMap {
		cliConfig.Env = append(cliConfig.Env, conf.EnvVarWithNodeFilters{
			EnvVar:      envVar,
			NodeFilters: nodeFilters,
		})
	}

	log.Tracef("EnvFilterMap: %+v", envFilterMap)
}
