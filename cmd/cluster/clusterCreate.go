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
package cluster

import (
	"fmt"
	"net/netip"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"

	cliutil "github.com/k3d-io/k3d/v5/cmd/util"
	cliconfig "github.com/k3d-io/k3d/v5/cmd/util/config"
	k3dCluster "github.com/k3d-io/k3d/v5/pkg/client"
	"github.com/k3d-io/k3d/v5/pkg/config"
	conf "github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/version"
)

const clusterCreateDescription = `
Create a new k3s cluster with containerized nodes (k3s in docker).
Every cluster will consist of one or more containers:
	- 1 (or more) server node container (k3s)
	- (optionally) 1 loadbalancer container as the entrypoint to the cluster (nginx)
	- (optionally) 1 (or more) agent node containers (k3s)
`

/*
 * Viper for configuration handling
 * we use two different instances of Viper here to handle
 * - cfgViper: "static" configuration
 * - ppViper: "pre-processed" configuration, where CLI input has to be pre-processed
 *             to be treated as part of the SImpleConfig
 */
var (
	cfgViper = viper.New()
	ppViper  = viper.New()
)

func initConfig() error {
	// Viper for pre-processed config options
	ppViper.SetEnvPrefix("K3D")
	ppViper.AutomaticEnv()
	ppViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if l.Log().GetLevel() >= logrus.DebugLevel {
		c, _ := yaml.Marshal(ppViper.AllSettings())
		l.Log().Debugf("Additional CLI Configuration:\n%s", c)
	}

	return cliconfig.InitViperWithConfigFile(cfgViper, ppViper.GetString("config"))
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
			return initConfig()
		},
		Run: func(cmd *cobra.Command, args []string) {
			/*************************
			 * Compute Configuration *
			 *************************/
			simpleCfg, err := config.SimpleConfigFromViper(cfgViper)
			if err != nil {
				l.Log().Fatalln(err)
			}

			l.Log().Debugf("========== Simple Config ==========\n%+v\n==========================\n", simpleCfg)

			simpleCfg, err = applyCLIOverrides(simpleCfg)
			if err != nil {
				l.Log().Fatalf("Failed to apply CLI overrides: %+v", err)
			}

			l.Log().Debugf("========== Merged Simple Config ==========\n%+v\n==========================\n", simpleCfg)

			/**************************************
			 * Transform, Process & Validate Configuration *
			 **************************************/

			// Set the name
			if len(args) != 0 {
				simpleCfg.Name = args[0]
			}

			if err := config.ProcessSimpleConfig(&simpleCfg); err != nil {
				l.Log().Fatalf("error processing/sanitizing simple config: %v", err)
			}

			clusterConfig, err := config.TransformSimpleToClusterConfig(cmd.Context(), runtimes.SelectedRuntime, simpleCfg, ppViper.GetString("config"))
			if err != nil {
				l.Log().Fatalln(err)
			}
			l.Log().Debugf("===== Merged Cluster Config =====\n%+v\n===== ===== =====\n", clusterConfig)

			clusterConfig, err = config.ProcessClusterConfig(*clusterConfig)
			if err != nil {
				l.Log().Fatalf("error processing cluster configuration: %v", err)
			}

			if err := config.ValidateClusterConfig(cmd.Context(), runtimes.SelectedRuntime, *clusterConfig); err != nil {
				l.Log().Fatalln("Failed Cluster Configuration Validation: ", err)
			}

			/**************************************
			 * Create cluster if it doesn't exist *
			 **************************************/

			// check if a cluster with that name exists already
			if _, err := k3dCluster.ClusterGet(cmd.Context(), runtimes.SelectedRuntime, &clusterConfig.Cluster); err == nil {
				l.Log().Fatalf("Failed to create cluster '%s' because a cluster with that name already exists", clusterConfig.Cluster.Name)
			}

			// create cluster
			if clusterConfig.KubeconfigOpts.UpdateDefaultKubeconfig {
				l.Log().Debugln("'--kubeconfig-update-default set: enabling wait-for-server")
				clusterConfig.ClusterCreateOpts.WaitForServer = true
			}
			//if err := k3dCluster.ClusterCreate(cmd.Context(), runtimes.SelectedRuntime, &clusterConfig.Cluster, &clusterConfig.ClusterCreateOpts); err != nil {
			if err := k3dCluster.ClusterRun(cmd.Context(), runtimes.SelectedRuntime, clusterConfig); err != nil {
				// rollback if creation failed
				l.Log().Errorln(err)
				if simpleCfg.Options.K3dOptions.NoRollback { // TODO: move rollback mechanics to pkg/
					l.Log().Fatalln("Cluster creation FAILED, rollback deactivated.")
				}
				// rollback if creation failed
				l.Log().Errorln("Failed to create cluster >>> Rolling Back")
				if err := k3dCluster.ClusterDelete(cmd.Context(), runtimes.SelectedRuntime, &clusterConfig.Cluster, k3d.ClusterDeleteOpts{SkipRegistryCheck: true}); err != nil {
					l.Log().Errorln(err)
					l.Log().Fatalln("Cluster creation FAILED, also FAILED to rollback changes!")
				}
				l.Log().Fatalln("Cluster creation FAILED, all changes have been rolled back!")
			}
			l.Log().Infof("Cluster '%s' created successfully!", clusterConfig.Cluster.Name)

			/**************
			 * Kubeconfig *
			 **************/

			if !clusterConfig.KubeconfigOpts.UpdateDefaultKubeconfig && clusterConfig.KubeconfigOpts.SwitchCurrentContext {
				l.Log().Infoln("--kubeconfig-update-default=false --> sets --kubeconfig-switch-context=false")
				clusterConfig.KubeconfigOpts.SwitchCurrentContext = false
			}

			if clusterConfig.KubeconfigOpts.UpdateDefaultKubeconfig {
				l.Log().Debugf("Updating default kubeconfig with a new context for cluster %s", clusterConfig.Cluster.Name)
				if _, err := k3dCluster.KubeconfigGetWrite(cmd.Context(), runtimes.SelectedRuntime, &clusterConfig.Cluster, "", &k3dCluster.WriteKubeConfigOptions{UpdateExisting: true, OverwriteExisting: false, UpdateCurrentContext: simpleCfg.Options.KubeconfigOptions.SwitchCurrentContext}); err != nil {
					l.Log().Warningln(err)
				}
			}

			/*****************
			 * User Feedback *
			 *****************/

			// print information on how to use the cluster with kubectl
			l.Log().Infoln("You can now use it like this:")
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

	cmd.Flags().StringP("config", "c", "", "Path of a config file to use")
	_ = ppViper.BindPFlag("config", cmd.Flags().Lookup("config"))
	if err := cmd.MarkFlagFilename("config", "yaml", "yml"); err != nil {
		l.Log().Fatalln("Failed to mark flag 'config' as filename flag")
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

	cmd.Flags().StringArrayP("env", "e", nil, "Add environment variables to nodes (Format: `KEY[=VALUE][@NODEFILTER[;NODEFILTER...]]`\n - Example: `k3d cluster create --agents 2 -e \"HTTP_PROXY=my.proxy.com@server:0\" -e \"SOME_KEY=SOME_VAL@server:0\"`")
	_ = ppViper.BindPFlag("cli.env", cmd.Flags().Lookup("env"))

	cmd.Flags().StringArrayP("volume", "v", nil, "Mount volumes into the nodes (Format: `[SOURCE:]DEST[@NODEFILTER[;NODEFILTER...]]`\n - Example: `k3d cluster create --agents 2 -v /my/path@agent:0,1 -v /tmp/test:/tmp/other@server:0`")
	_ = ppViper.BindPFlag("cli.volumes", cmd.Flags().Lookup("volume"))

	cmd.Flags().StringArrayP("port", "p", nil, "Map ports from the node containers (via the serverlb) to the host (Format: `[HOST:][HOSTPORT:]CONTAINERPORT[/PROTOCOL][@NODEFILTER]`)\n - Example: `k3d cluster create --agents 2 -p 8080:80@agent:0 -p 8081@agent:1`")
	_ = ppViper.BindPFlag("cli.ports", cmd.Flags().Lookup("port"))

	cmd.Flags().StringArrayP("k3s-node-label", "", nil, "Add label to k3s node (Format: `KEY[=VALUE][@NODEFILTER[;NODEFILTER...]]`\n - Example: `k3d cluster create --agents 2 --k3s-node-label \"my.label@agent:0,1\" --k3s-node-label \"other.label=somevalue@server:0\"`")
	_ = ppViper.BindPFlag("cli.k3s-node-labels", cmd.Flags().Lookup("k3s-node-label"))

	cmd.Flags().StringArrayP("runtime-label", "", nil, "Add label to container runtime (Format: `KEY[=VALUE][@NODEFILTER[;NODEFILTER...]]`\n - Example: `k3d cluster create --agents 2 --runtime-label \"my.label@agent:0,1\" --runtime-label \"other.label=somevalue@server:0\"`")
	_ = ppViper.BindPFlag("cli.runtime-labels", cmd.Flags().Lookup("runtime-label"))

	cmd.Flags().StringArrayP("runtime-ulimit", "", nil, "Add ulimit to container runtime (Format: `NAME[=SOFT]:[HARD]`\n - Example: `k3d cluster create --agents 2 --runtime-ulimit \"nofile=1024:1024\" --runtime-ulimit \"noproc=1024:1024\"`")
	_ = ppViper.BindPFlag("cli.runtime-ulimits", cmd.Flags().Lookup("runtime-ulimit"))

	cmd.Flags().String("registry-create", "", "Create a k3d-managed registry and connect it to the cluster (Format: `NAME[:HOST][:HOSTPORT]`\n - Example: `k3d cluster create --registry-create mycluster-registry:0.0.0.0:5432`")
	_ = ppViper.BindPFlag("cli.registries.create", cmd.Flags().Lookup("registry-create"))

	cmd.Flags().StringArray("host-alias", nil, "Add `ip:host[,host,...]` mappings")
	_ = ppViper.BindPFlag("hostaliases", cmd.Flags().Lookup("host-alias"))

	/* k3s */
	cmd.Flags().StringArray("k3s-arg", nil, "Additional args passed to k3s command (Format: `ARG@NODEFILTER[;@NODEFILTER]`)\n - Example: `k3d cluster create --k3s-arg \"--disable=traefik@server:0\"`")
	_ = ppViper.BindPFlag("cli.k3sargs", cmd.Flags().Lookup("k3s-arg"))

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
	cfgViper.SetDefault("image", fmt.Sprintf("%s:%s", k3d.DefaultK3sImageRepo, version.K3sVersion))

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

	cmd.Flags().String("gpus", "", "GPU devices to add to the cluster node containers ('all' to pass all GPUs) [From docker]")
	_ = cfgViper.BindPFlag("options.runtime.gpurequest", cmd.Flags().Lookup("gpus"))

	cmd.Flags().String("servers-memory", "", "Memory limit imposed on the server nodes [From docker]")
	_ = cfgViper.BindPFlag("options.runtime.serversmemory", cmd.Flags().Lookup("servers-memory"))

	cmd.Flags().String("agents-memory", "", "Memory limit imposed on the agents nodes [From docker]")
	_ = cfgViper.BindPFlag("options.runtime.agentsmemory", cmd.Flags().Lookup("agents-memory"))

	cmd.Flags().Bool("host-pid-mode", false, "Enable host pid mode of server(s) and agent(s)")
	_ = cfgViper.BindPFlag("options.runtime.hostpidmode", cmd.Flags().Lookup("host-pid-mode"))

	/* Image Importing */
	cmd.Flags().Bool("no-image-volume", false, "Disable the creation of a volume for importing images")
	_ = cfgViper.BindPFlag("options.k3d.disableimagevolume", cmd.Flags().Lookup("no-image-volume"))

	/* Registry */
	cmd.Flags().StringArray("registry-use", nil, "Connect to one or more k3d-managed registries running locally")
	_ = cfgViper.BindPFlag("registries.use", cmd.Flags().Lookup("registry-use"))

	cmd.Flags().String("registry-config", "", "Specify path to an extra registries.yaml file")
	_ = cfgViper.BindPFlag("registries.config", cmd.Flags().Lookup("registry-config"))
	if err := cmd.MarkFlagFilename("registry-config", "yaml", "yml"); err != nil {
		l.Log().Fatalln("Failed to mark flag 'config' as filename flag")
	}

	/* Loadbalancer / Proxy */
	cmd.Flags().StringSlice("lb-config-override", nil, "Use dotted YAML path syntax to override nginx loadbalancer settings")
	_ = cfgViper.BindPFlag("options.k3d.loadbalancer.configoverrides", cmd.Flags().Lookup("lb-config-override"))

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
			l.Log().Debugf("Overriding pre-defined kubeAPI Exposure Spec %+v with CLI argument %s", cfg.ExposeAPI, ppViper.GetString("cli.api-port"))
		}
		exposeAPI, err = cliutil.ParsePortExposureSpec(ppViper.GetString("cli.api-port"), k3d.DefaultAPIPort)
		if err != nil {
			return cfg, fmt.Errorf("failed to parse API Port spec: %w", err)
		}
	}

	// Set to random port if port is empty string
	if len(exposeAPI.Binding.HostPort) == 0 {
		var freePort string
		port, err := cliutil.GetFreePort()
		freePort = strconv.Itoa(port)
		if err != nil || port == 0 {
			l.Log().Warnf("Failed to get random free port: %+v", err)
			l.Log().Warnf("Falling back to internal port %s (may be blocked though)...", k3d.DefaultAPIPort)
			freePort = k3d.DefaultAPIPort
		}
		exposeAPI.Binding.HostPort = freePort
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
			l.Log().Fatalln(err)
		}

		if strings.Contains(volume, k3d.DefaultRegistriesFilePath) && (cfg.Registries.Create != nil || cfg.Registries.Config != "" || len(cfg.Registries.Use) != 0) {
			l.Log().Warnf("Seems like you're mounting a file at '%s' while also using a referenced registries config or k3d-managed registries: Your mounted file will probably be overwritten!", k3d.DefaultRegistriesFilePath)
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

	l.Log().Tracef("VolumeFilterMap: %+v", volumeFilterMap)

	// -> PORTS
	portFilterMap := make(map[string][]string, 1)
	for _, portFlag := range ppViper.GetStringSlice("cli.ports") {
		// split node filter from the specified volume
		portmap, filters, err := cliutil.SplitFiltersFromFlag(portFlag)
		if err != nil {
			l.Log().Fatalln(err)
		}

		// create new entry or append filter to existing entry
		if _, exists := portFilterMap[portmap]; exists {
			l.Log().Fatalln("Same Portmapping can not be used for multiple nodes")
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

	l.Log().Tracef("PortFilterMap: %+v", portFilterMap)

	// --k3s-node-label
	// k3sNodeLabelFilterMap will add k3s node label to applied node filters
	k3sNodeLabelFilterMap := make(map[string][]string, 1)
	for _, labelFlag := range ppViper.GetStringSlice("cli.k3s-node-labels") {
		// split node filter from the specified label
		label, nodeFilters, err := cliutil.SplitFiltersFromFlag(labelFlag)
		if err != nil {
			l.Log().Fatalln(err)
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

	l.Log().Tracef("K3sNodeLabelFilterMap: %+v", k3sNodeLabelFilterMap)

	// --runtime-label
	// runtimeLabelFilterMap will add container runtime label to applied node filters
	runtimeLabelFilterMap := make(map[string][]string, 1)
	for _, labelFlag := range ppViper.GetStringSlice("cli.runtime-labels") {
		// split node filter from the specified label
		label, nodeFilters, err := cliutil.SplitFiltersFromFlag(labelFlag)
		if err != nil {
			l.Log().Fatalln(err)
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

	l.Log().Tracef("RuntimeLabelFilterMap: %+v", runtimeLabelFilterMap)

	for _, ulimit := range ppViper.GetStringSlice("cli.runtime-ulimits") {
		cfg.Options.Runtime.Ulimits = append(cfg.Options.Runtime.Ulimits, *cliutil.ParseRuntimeUlimit[conf.Ulimit](ulimit))
	}

	// --env
	// envFilterMap will add container env vars to applied node filters
	envFilterMap := make(map[string][]string, 1)
	for _, envFlag := range ppViper.GetStringSlice("cli.env") {
		// split node filter from the specified env var
		env, filters, err := cliutil.SplitFiltersFromFlag(envFlag)
		if err != nil {
			l.Log().Fatalln(err)
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

	l.Log().Tracef("EnvFilterMap: %+v", envFilterMap)

	// --k3s-arg
	argFilterMap := make(map[string][]string, 1)
	for _, argFlag := range ppViper.GetStringSlice("cli.k3sargs") {
		// split node filter from the specified arg
		arg, filters, err := cliutil.SplitFiltersFromFlag(argFlag)
		if err != nil {
			l.Log().Fatalln(err)
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

	// --registry-create
	if ppViper.IsSet("cli.registries.create") {
		flagvalue := ppViper.GetString("cli.registries.create")
		fvSplit := strings.SplitN(flagvalue, ":", 2)
		if cfg.Registries.Create == nil {
			cfg.Registries.Create = &conf.SimpleConfigRegistryCreateConfig{}
		}
		cfg.Registries.Create.Name = fvSplit[0]
		if len(fvSplit) > 1 {
			exposeAPI, err = cliutil.ParseRegistryPortExposureSpec(fvSplit[1])
			if err != nil {
				return cfg, fmt.Errorf("failed to registry port spec: %w", err)
			}
			cfg.Registries.Create.Host = exposeAPI.Host
			cfg.Registries.Create.HostPort = exposeAPI.Binding.HostPort
		}
	}

	// --host-alias
	hostAliasFlags := ppViper.GetStringSlice("hostaliases")
	if len(hostAliasFlags) > 0 {
		for _, ha := range hostAliasFlags {
			// split on :
			s := strings.Split(ha, ":")
			if len(s) != 2 {
				return cfg, fmt.Errorf("invalid format of host-alias %s (exactly one ':' allowed)", ha)
			}

			// validate IP
			ip, err := netip.ParseAddr(s[0])
			if err != nil {
				return cfg, fmt.Errorf("invalid IP '%s' in host-alias '%s': %w", s[0], ha, err)
			}

			// hostnames
			hostnames := strings.Split(s[1], ",")
			for _, hostname := range hostnames {
				if err := k3dCluster.ValidateHostname(hostname); err != nil {
					return cfg, fmt.Errorf("invalid hostname '%s' in host-alias '%s': %w", hostname, ha, err)
				}
			}

			cfg.HostAliases = append(cfg.HostAliases, k3d.HostAlias{
				IP:        ip.String(),
				Hostnames: hostnames,
			})
		}
	}

	return cfg, nil
}
