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
package cluster

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"

	cliutil "github.com/rancher/k3d/v4/cmd/util"
	k3dCluster "github.com/rancher/k3d/v4/pkg/client"
	"github.com/rancher/k3d/v4/pkg/config"
	conf "github.com/rancher/k3d/v4/pkg/config/v1alpha3"
	"github.com/rancher/k3d/v4/pkg/runtimes"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	"github.com/rancher/k3d/v4/version"

	log "github.com/sirupsen/logrus"
)

var configFile string

const clusterCreateDescription = `
Create a new k3s cluster with containerized nodes (k3s in docker).
Every cluster will consist of one or more containers:
	- 1 (or more) server node container (k3s)
	- (optionally) 1 loadbalancer container as the entrypoint to the cluster (nginx)
	- (optionally) 1 (or more) agent node containers (k3s)
`

var cfgViper = viper.New()
var ppViper = viper.New()

func initConfig() {

	// Viper for pre-processed config options
	ppViper.SetEnvPrefix("K3D")

	// viper for the general config (file, env and non pre-processed flags)
	cfgViper.SetEnvPrefix("K3D")
	cfgViper.AutomaticEnv()

	cfgViper.SetConfigType("yaml")

	// Set config file, if specified
	if configFile != "" {
		cfgViper.SetConfigFile(configFile)

		if _, err := os.Stat(configFile); err != nil {
			log.Fatalf("Failed to stat config file %s: %+v", configFile, err)
		}

		// try to read config into memory (viper map structure)
		if err := cfgViper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				log.Fatalf("Config file %s not found: %+v", configFile, err)
			}
			// config file found but some other error happened
			log.Fatalf("Failed to read config file %s: %+v", configFile, err)
		}

		schema, err := config.GetSchemaByVersion(cfgViper.GetString("apiVersion"))
		if err != nil {
			log.Fatalf("Cannot validate config file %s: %+v", configFile, err)
		}

		if err := config.ValidateSchemaFile(configFile, schema); err != nil {
			log.Fatalf("Schema Validation failed for config file %s: %+v", configFile, err)
		}

		log.Infof("Using config file %s (%s#%s)", cfgViper.ConfigFileUsed(), strings.ToLower(cfgViper.GetString("apiVersion")), strings.ToLower(cfgViper.GetString("kind")))
	}
	if log.GetLevel() >= log.DebugLevel {
		c, _ := yaml.Marshal(cfgViper.AllSettings())
		log.Debugf("Configuration:\n%s", c)

		c, _ = yaml.Marshal(ppViper.AllSettings())
		log.Debugf("Additional CLI Configuration:\n%s", c)
	}
}

// NewCmdClusterCreate returns a new cobra command
func NewCmdClusterCreate() *cobra.Command {

	// create new command
	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a new cluster",
		Long:  clusterCreateDescription,
		Args:  cobra.RangeArgs(0, 1), // exactly one cluster name can be set (default: k3d.DefaultClusterName)
		PreRunE: func(cmd *cobra.Command, args []string) error {
			initConfig()
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {

			/*************************
			 * Compute Configuration *
			 *************************/
			if cfgViper.GetString("apiversion") == "" {
				cfgViper.Set("apiversion", config.DefaultConfigApiVersion)
			}
			if cfgViper.GetString("kind") == "" {
				cfgViper.Set("kind", "Simple")
			}
			cfg, err := config.FromViper(cfgViper)
			if err != nil {
				log.Fatalln(err)
			}

			if cfg.GetAPIVersion() != config.DefaultConfigApiVersion {
				log.Warnf("Default config apiVersion is '%s', but you're using '%s': consider migrating.", config.DefaultConfigApiVersion, cfg.GetAPIVersion())
				cfg, err = config.Migrate(cfg, config.DefaultConfigApiVersion)
				if err != nil {
					log.Fatalln(err)
				}
			}

			simpleCfg := cfg.(conf.SimpleConfig)

			log.Debugf("========== Simple Config ==========\n%+v\n==========================\n", simpleCfg)

			simpleCfg, err = applyCLIOverrides(simpleCfg)
			if err != nil {
				log.Fatalf("Failed to apply CLI overrides: %+v", err)
			}

			log.Debugf("========== Merged Simple Config ==========\n%+v\n==========================\n", simpleCfg)

			/**************************************
			 * Transform, Process & Validate Configuration *
			 **************************************/

			// Set the name
			if len(args) != 0 {
				simpleCfg.Name = args[0]
			}

			clusterConfig, err := config.TransformSimpleToClusterConfig(cmd.Context(), runtimes.SelectedRuntime, simpleCfg)
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
				if simpleCfg.Options.K3dOptions.NoRollback { // TODO: move rollback mechanics to pkg/
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
				if _, err := k3dCluster.KubeconfigGetWrite(cmd.Context(), runtimes.SelectedRuntime, &clusterConfig.Cluster, "", &k3dCluster.WriteKubeConfigOptions{UpdateExisting: true, OverwriteExisting: false, UpdateCurrentContext: simpleCfg.Options.KubeconfigOptions.SwitchCurrentContext}); err != nil {
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

	/***************
	 * Config File *
	 ***************/

	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Path of a config file to use")
	if err := cobra.MarkFlagFilename(cmd.Flags(), "config", "yaml", "yml"); err != nil {
		log.Fatalln("Failed to mark flag 'config' as filename flag")
	}

	/***********************
	 * Pre-Processed Flags *
	 ***********************
	 *
	 * Flags that have a different style in the CLI than their internal representation.
	 * Also, we cannot set (viper) default values just here for those.
	 * Example:
	 *   CLI: `--api-port 0.0.0.0:6443`
	 *   Config File:
	 *	   exposeAPI:
	 *			 hostIP: 0.0.0.0
	 *       port: 6443
	 *
	 * Note: here we also use Slice-type flags instead of Array because of https://github.com/spf13/viper/issues/380
	 */

	cmd.Flags().String("api-port", "", "Specify the Kubernetes API server port exposed on the LoadBalancer (Format: `[HOST:]HOSTPORT`)\n - Example: `k3d cluster create --servers 3 --api-port 0.0.0.0:6550`")
	_ = ppViper.BindPFlag("cli.api-port", cmd.Flags().Lookup("api-port"))

	cmd.Flags().StringArrayP("env", "e", nil, "Add environment variables to nodes (Format: `KEY[=VALUE][@NODEFILTER[;NODEFILTER...]]`\n - Example: `k3d cluster create --agents 2 -e \"HTTP_PROXY=my.proxy.com@server[0]\" -e \"SOME_KEY=SOME_VAL@server[0]\"`")
	_ = ppViper.BindPFlag("cli.env", cmd.Flags().Lookup("env"))

	cmd.Flags().StringArrayP("volume", "v", nil, "Mount volumes into the nodes (Format: `[SOURCE:]DEST[@NODEFILTER[;NODEFILTER...]]`\n - Example: `k3d cluster create --agents 2 -v /my/path@agent[0,1] -v /tmp/test:/tmp/other@server[0]`")
	_ = ppViper.BindPFlag("cli.volumes", cmd.Flags().Lookup("volume"))

	cmd.Flags().StringArrayP("port", "p", nil, "Map ports from the node containers to the host (Format: `[HOST:][HOSTPORT:]CONTAINERPORT[/PROTOCOL][@NODEFILTER]`)\n - Example: `k3d cluster create --agents 2 -p 8080:80@agent[0] -p 8081@agent[1]`")
	_ = ppViper.BindPFlag("cli.ports", cmd.Flags().Lookup("port"))

	cmd.Flags().StringArrayP("k3s-node-label", "", nil, "Add label to k3s node (Format: `KEY[=VALUE][@NODEFILTER[;NODEFILTER...]]`\n - Example: `k3d cluster create --agents 2 --k3s-node-label \"my.label@agent[0,1]\" --k3s-node-label \"other.label=somevalue@server[0]\"`")
	_ = ppViper.BindPFlag("cli.k3s-node-labels", cmd.Flags().Lookup("k3s-node-label"))

	cmd.Flags().StringArrayP("runtime-label", "", nil, "Add label to container runtime (Format: `KEY[=VALUE][@NODEFILTER[;NODEFILTER...]]`\n - Example: `k3d cluster create --agents 2 --runtime-label \"my.label@agent[0,1]\" --runtime-label \"other.label=somevalue@server[0]\"`")
	_ = ppViper.BindPFlag("cli.runtime-labels", cmd.Flags().Lookup("runtime-label"))

	/* k3s */
	cmd.Flags().StringArray("k3s-arg", nil, "Additional args passed to k3s command (Format: `ARG@NODEFILTER[;@NODEFILTER]`)\n - Example: `k3d cluster create --k3s-arg \"--disable=traefik@server[0]\"")
	_ = cfgViper.BindPFlag("cli.k3sargs", cmd.Flags().Lookup("k3s-arg"))

	/******************
	 * "Normal" Flags *
	 ******************
	 *
	 * No pre-processing needed on CLI level.
	 * Bound to Viper config value.
	 * Default Values set via Viper.
	 */

	cmd.Flags().IntP("servers", "s", 0, "Specify how many servers you want to create")
	_ = cfgViper.BindPFlag("servers", cmd.Flags().Lookup("servers"))
	cfgViper.SetDefault("servers", 1)

	cmd.Flags().IntP("agents", "a", 0, "Specify how many agents you want to create")
	_ = cfgViper.BindPFlag("agents", cmd.Flags().Lookup("agents"))
	cfgViper.SetDefault("agents", 0)

	cmd.Flags().StringP("image", "i", "", "Specify k3s image that you want to use for the nodes")
	_ = cfgViper.BindPFlag("image", cmd.Flags().Lookup("image"))
	cfgViper.SetDefault("image", fmt.Sprintf("%s:%s", k3d.DefaultK3sImageRepo, version.GetK3sVersion(false)))

	cmd.Flags().String("network", "", "Join an existing network")
	_ = cfgViper.BindPFlag("network", cmd.Flags().Lookup("network"))

	cmd.Flags().String("subnet", "", "[Experimental: IPAM] Define a subnet for the newly created container network (Example: `172.28.0.0/16`)")
	_ = cfgViper.BindPFlag("subnet", cmd.Flags().Lookup("subnet"))

	cmd.Flags().String("token", "", "Specify a cluster token. By default, we generate one.")
	_ = cfgViper.BindPFlag("token", cmd.Flags().Lookup("token"))

	cmd.Flags().Bool("wait", true, "Wait for the server(s) to be ready before returning. Use '--timeout DURATION' to not wait forever.")
	_ = cfgViper.BindPFlag("options.k3d.wait", cmd.Flags().Lookup("wait"))

	cmd.Flags().Duration("timeout", 0*time.Second, "Rollback changes if cluster couldn't be created in specified duration.")
	_ = cfgViper.BindPFlag("options.k3d.timeout", cmd.Flags().Lookup("timeout"))

	cmd.Flags().Bool("kubeconfig-update-default", true, "Directly update the default kubeconfig with the new cluster's context")
	_ = cfgViper.BindPFlag("options.kubeconfig.updatedefaultkubeconfig", cmd.Flags().Lookup("kubeconfig-update-default"))

	cmd.Flags().Bool("kubeconfig-switch-context", true, "Directly switch the default kubeconfig's current-context to the new cluster's context (requires --kubeconfig-update-default)")
	_ = cfgViper.BindPFlag("options.kubeconfig.switchcurrentcontext", cmd.Flags().Lookup("kubeconfig-switch-context"))

	cmd.Flags().Bool("no-lb", false, "Disable the creation of a LoadBalancer in front of the server nodes")
	_ = cfgViper.BindPFlag("options.k3d.disableloadbalancer", cmd.Flags().Lookup("no-lb"))

	cmd.Flags().Bool("no-rollback", false, "Disable the automatic rollback actions, if anything goes wrong")
	_ = cfgViper.BindPFlag("options.k3d.disablerollback", cmd.Flags().Lookup("no-rollback"))

	cmd.Flags().Bool("no-hostip", false, "Disable the automatic injection of the Host IP as 'host.k3d.internal' into the containers and CoreDNS")
	_ = cfgViper.BindPFlag("options.k3d.disablehostipinjection", cmd.Flags().Lookup("no-hostip"))

	cmd.Flags().String("gpus", "", "GPU devices to add to the cluster node containers ('all' to pass all GPUs) [From docker]")
	_ = cfgViper.BindPFlag("options.runtime.gpurequest", cmd.Flags().Lookup("gpus"))

	cmd.Flags().String("servers-memory", "", "Memory limit imposed on the server nodes [From docker]")
	_ = cfgViper.BindPFlag("options.runtime.serversmemory", cmd.Flags().Lookup("servers-memory"))

	cmd.Flags().String("agents-memory", "", "Memory limit imposed on the agents nodes [From docker]")
	_ = cfgViper.BindPFlag("options.runtime.agentsmemory", cmd.Flags().Lookup("agents-memory"))

	/* Image Importing */
	cmd.Flags().Bool("no-image-volume", false, "Disable the creation of a volume for importing images")
	_ = cfgViper.BindPFlag("options.k3d.disableimagevolume", cmd.Flags().Lookup("no-image-volume"))

	/* Registry */
	cmd.Flags().StringArray("registry-use", nil, "Connect to one or more k3d-managed registries running locally")
	_ = cfgViper.BindPFlag("registries.use", cmd.Flags().Lookup("registry-use"))

	cmd.Flags().Bool("registry-create", false, "Create a k3d-managed registry and connect it to the cluster")
	_ = cfgViper.BindPFlag("registries.create", cmd.Flags().Lookup("registry-create"))

	cmd.Flags().String("registry-config", "", "Specify path to an extra registries.yaml file")
	_ = cfgViper.BindPFlag("registries.config", cmd.Flags().Lookup("registry-config"))

	/* Subcommands */

	// done
	return cmd
}

func applyCLIOverrides(cfg conf.SimpleConfig) (conf.SimpleConfig, error) {

	/****************************
	 * Parse and validate flags *
	 ****************************/

	// -> API-PORT
	// parse the port mapping
	var (
		err       error
		exposeAPI *k3d.ExposureOpts
	)

	// Apply config file values as defaults
	exposeAPI = &k3d.ExposureOpts{
		PortMapping: nat.PortMapping{
			Binding: nat.PortBinding{
				HostIP:   cfg.ExposeAPI.HostIP,
				HostPort: cfg.ExposeAPI.HostPort,
			},
		},
		Host: cfg.ExposeAPI.Host,
	}

	// Overwrite if cli arg is set
	if ppViper.IsSet("cli.api-port") {
		if cfg.ExposeAPI.HostPort != "" {
			log.Debugf("Overriding pre-defined kubeAPI Exposure Spec %+v with CLI argument %s", cfg.ExposeAPI, ppViper.GetString("cli.api-port"))
		}
		exposeAPI, err = cliutil.ParsePortExposureSpec(ppViper.GetString("cli.api-port"), k3d.DefaultAPIPort)
		if err != nil {
			return cfg, err
		}
	}

	// Set to random port if port is empty string
	if len(exposeAPI.Binding.HostPort) == 0 {
		exposeAPI, err = cliutil.ParsePortExposureSpec("random", k3d.DefaultAPIPort)
		if err != nil {
			return cfg, err
		}
	}

	cfg.ExposeAPI = conf.SimpleExposureOpts{
		Host:     exposeAPI.Host,
		HostIP:   exposeAPI.Binding.HostIP,
		HostPort: exposeAPI.Binding.HostPort,
	}

	// -> VOLUMES
	// volumeFilterMap will map volume mounts to applied node filters
	volumeFilterMap := make(map[string][]string, 1)
	for _, volumeFlag := range ppViper.GetStringSlice("cli.volumes") {

		// split node filter from the specified volume
		volume, filters, err := cliutil.SplitFiltersFromFlag(volumeFlag)
		if err != nil {
			log.Fatalln(err)
		}

		if strings.Contains(volume, k3d.DefaultRegistriesFilePath) && (cfg.Registries.Create || cfg.Registries.Config != "" || len(cfg.Registries.Use) != 0) {
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
		cfg.Volumes = append(cfg.Volumes, conf.VolumeWithNodeFilters{
			Volume:      volume,
			NodeFilters: nodeFilters,
		})
	}

	log.Tracef("VolumeFilterMap: %+v", volumeFilterMap)

	// -> PORTS
	portFilterMap := make(map[string][]string, 1)
	for _, portFlag := range ppViper.GetStringSlice("cli.ports") {
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
		cfg.Ports = append(cfg.Ports, conf.PortWithNodeFilters{
			Port:        port,
			NodeFilters: nodeFilters,
		})
	}

	log.Tracef("PortFilterMap: %+v", portFilterMap)

	// --k3s-node-label
	// k3sNodeLabelFilterMap will add k3s node label to applied node filters
	k3sNodeLabelFilterMap := make(map[string][]string, 1)
	for _, labelFlag := range ppViper.GetStringSlice("cli.k3s-node-labels") {

		// split node filter from the specified label
		label, nodeFilters, err := cliutil.SplitFiltersFromFlag(labelFlag)
		if err != nil {
			log.Fatalln(err)
		}

		// create new entry or append filter to existing entry
		if _, exists := k3sNodeLabelFilterMap[label]; exists {
			k3sNodeLabelFilterMap[label] = append(k3sNodeLabelFilterMap[label], nodeFilters...)
		} else {
			k3sNodeLabelFilterMap[label] = nodeFilters
		}
	}

	for label, nodeFilters := range k3sNodeLabelFilterMap {
		cfg.Options.K3sOptions.NodeLabels = append(cfg.Options.K3sOptions.NodeLabels, conf.LabelWithNodeFilters{
			Label:       label,
			NodeFilters: nodeFilters,
		})
	}

	log.Tracef("K3sNodeLabelFilterMap: %+v", k3sNodeLabelFilterMap)

	// --runtime-label
	// runtimeLabelFilterMap will add container runtime label to applied node filters
	runtimeLabelFilterMap := make(map[string][]string, 1)
	for _, labelFlag := range ppViper.GetStringSlice("cli.runtime-labels") {

		// split node filter from the specified label
		label, nodeFilters, err := cliutil.SplitFiltersFromFlag(labelFlag)
		if err != nil {
			log.Fatalln(err)
		}

		cliutil.ValidateRuntimeLabelKey(strings.Split(label, "=")[0])

		// create new entry or append filter to existing entry
		if _, exists := runtimeLabelFilterMap[label]; exists {
			runtimeLabelFilterMap[label] = append(runtimeLabelFilterMap[label], nodeFilters...)
		} else {
			runtimeLabelFilterMap[label] = nodeFilters
		}
	}

	for label, nodeFilters := range runtimeLabelFilterMap {
		cfg.Options.Runtime.Labels = append(cfg.Options.Runtime.Labels, conf.LabelWithNodeFilters{
			Label:       label,
			NodeFilters: nodeFilters,
		})
	}

	log.Tracef("RuntimeLabelFilterMap: %+v", runtimeLabelFilterMap)

	// --env
	// envFilterMap will add container env vars to applied node filters
	envFilterMap := make(map[string][]string, 1)
	for _, envFlag := range ppViper.GetStringSlice("cli.env") {

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
		cfg.Env = append(cfg.Env, conf.EnvVarWithNodeFilters{
			EnvVar:      envVar,
			NodeFilters: nodeFilters,
		})
	}

	log.Tracef("EnvFilterMap: %+v", envFilterMap)

	// --k3s-arg
	argFilterMap := make(map[string][]string, 1)
	for _, argFlag := range ppViper.GetStringSlice("cli.k3sargs") {

		// split node filter from the specified arg
		arg, filters, err := cliutil.SplitFiltersFromFlag(argFlag)
		if err != nil {
			log.Fatalln(err)
		}

		// create new entry or append filter to existing entry
		if _, exists := argFilterMap[arg]; exists {
			argFilterMap[arg] = append(argFilterMap[arg], filters...)
		} else {
			argFilterMap[arg] = filters
		}
	}

	for arg, nodeFilters := range argFilterMap {
		cfg.Options.K3sOptions.ExtraArgs = append(cfg.Options.K3sOptions.ExtraArgs, conf.K3sArgWithNodeFilters{
			Arg:         arg,
			NodeFilters: nodeFilters,
		})
	}

	return cfg, nil
}
