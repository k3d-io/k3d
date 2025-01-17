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
package client

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/imdario/mergo"
	copystruct "github.com/mitchellh/copystructure"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/yaml"

	wharfie "github.com/rancher/wharfie/pkg/registries"

	"github.com/k3d-io/k3d/v5/pkg/actions"
	config "github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	k3drt "github.com/k3d-io/k3d/v5/pkg/runtimes"
	runtimeErr "github.com/k3d-io/k3d/v5/pkg/runtimes/errors"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/pkg/types/k3s"
	"github.com/k3d-io/k3d/v5/pkg/util"
	goyaml "gopkg.in/yaml.v2"
)

// ClusterRun orchestrates the steps of cluster creation, configuration and starting
func ClusterRun(ctx context.Context, runtime k3drt.Runtime, clusterConfig *config.ClusterConfig) error {
	/*
	 * Step 0: (Infrastructure) Preparation
	 */
	if err := ClusterPrep(ctx, runtime, clusterConfig); err != nil {
		return fmt.Errorf("Failed Cluster Preparation: %+v", err)
	}

	// Create tools-node for later steps

	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		_, err := EnsureToolsNode(ctx, runtime, &clusterConfig.Cluster)
		return err
	})

	/*
	 * Step 1: Create Containers
	 */
	if err := ClusterCreate(ctx, runtime, &clusterConfig.Cluster, &clusterConfig.ClusterCreateOpts); err != nil {
		return fmt.Errorf("failed Cluster Creation: %+v", err)
	}

	// Wait for tools node to be available
	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to ensure tools node: %w", err)
	}

	/*
	 * Step 2: Pre-Start Configuration
	 */
	// Gather Environment information, e.g. the host gateway address
	envInfo, err := GatherEnvironmentInfo(ctx, runtime, &clusterConfig.Cluster)
	if err != nil {
		return fmt.Errorf("failed to gather environment information used for cluster creation: %w", err)
	}

	/*
	 * Step 3: Start Containers
	 */
	if err := ClusterStart(ctx, runtime, &clusterConfig.Cluster, k3d.ClusterStartOpts{
		WaitForServer:   clusterConfig.ClusterCreateOpts.WaitForServer,
		Timeout:         clusterConfig.ClusterCreateOpts.Timeout, // TODO: here we should consider the time used so far
		NodeHooks:       clusterConfig.ClusterCreateOpts.NodeHooks,
		EnvironmentInfo: envInfo,
		Intent:          k3d.IntentClusterCreate,
		HostAliases:     clusterConfig.ClusterCreateOpts.HostAliases,
	}); err != nil {
		return fmt.Errorf("Failed Cluster Start: %+v", err)
	}

	/*
	 * Post-Start Configuration
	 */
	/**********************************
	 * Additional Cluster Preparation *
	 **********************************/

	// create the registry hosting configmap
	if len(clusterConfig.ClusterCreateOpts.Registries.Use) > 0 {
		if err := prepCreateLocalRegistryHostingConfigMap(ctx, runtime, &clusterConfig.Cluster); err != nil {
			l.Log().Warnf("Failed to create LocalRegistryHosting ConfigMap: %+v", err)
		}
	}

	return nil
}

// ClusterPrep takes care of the steps required before creating/starting the cluster containers
func ClusterPrep(ctx context.Context, runtime k3drt.Runtime, clusterConfig *config.ClusterConfig) error {
	/*
	 * Set up contexts
	 * Used for (early) termination (across API boundaries)
	 */
	clusterPrepCtx := ctx
	if clusterConfig.ClusterCreateOpts.Timeout > 0*time.Second {
		var cancelClusterPrepCtx context.CancelFunc
		clusterPrepCtx, cancelClusterPrepCtx = context.WithTimeout(ctx, clusterConfig.ClusterCreateOpts.Timeout)
		defer cancelClusterPrepCtx()
	}

	/*
	 * Step 0: Pre-Pull Images
	 */
	// TODO: ClusterPrep: add image pre-pulling step

	/*
	 * Step 1: Network
	 */
	if err := ClusterPrepNetwork(clusterPrepCtx, runtime, &clusterConfig.Cluster, &clusterConfig.ClusterCreateOpts); err != nil {
		return fmt.Errorf("Failed Network Preparation: %+v", err)
	}

	/*
	 * Step 2: Volume(s)
	 */
	if !clusterConfig.ClusterCreateOpts.DisableImageVolume {
		if err := ClusterPrepImageVolume(ctx, runtime, &clusterConfig.Cluster, &clusterConfig.ClusterCreateOpts); err != nil {
			return fmt.Errorf("Failed Image Volume Preparation: %+v", err)
		}
	}

	/*
	 * Step 3: Registries
	 */

	// Ensure referenced registries
	for _, reg := range clusterConfig.ClusterCreateOpts.Registries.Use {
		l.Log().Debugf("Trying to find registry %s", reg.Host)
		regNode, err := runtime.GetNode(ctx, &k3d.Node{Name: reg.Host})
		if err != nil {
			return fmt.Errorf("Failed to find registry node '%s': %+v", reg.Host, err)
		}
		regFromNode, err := RegistryFromNode(regNode)
		if err != nil {
			return fmt.Errorf("failed to translate node to registry spec: %w", err)
		}
		*reg = *regFromNode
	}

	// Create managed registry bound to this cluster
	if clusterConfig.ClusterCreateOpts.Registries.Create != nil {
		registryNode, err := RegistryCreate(ctx, runtime, clusterConfig.ClusterCreateOpts.Registries.Create)
		if err != nil {
			return fmt.Errorf("Failed to create registry: %+v", err)
		}

		clusterConfig.Cluster.Nodes = append(clusterConfig.Cluster.Nodes, registryNode)

		clusterConfig.ClusterCreateOpts.Registries.Use = append(clusterConfig.ClusterCreateOpts.Registries.Use, clusterConfig.ClusterCreateOpts.Registries.Create)
	}

	// Use existing registries (including the new one, if created)
	l.Log().Tracef("Using Registries: %+v", clusterConfig.ClusterCreateOpts.Registries.Use)

	var registryConfig *wharfie.Registry

	if len(clusterConfig.ClusterCreateOpts.Registries.Use) > 0 {
		// ensure that all selected registries exist and connect them to the cluster network
		for _, externalReg := range clusterConfig.ClusterCreateOpts.Registries.Use {
			regNode, err := runtime.GetNode(ctx, &k3d.Node{Name: externalReg.Host})
			if err != nil {
				return fmt.Errorf("Failed to find registry node '%s': %+v", externalReg.Host, err)
			}
			if err := RegistryConnectNetworks(ctx, runtime, regNode, []string{clusterConfig.Cluster.Network.Name}); err != nil {
				return fmt.Errorf("Failed to connect registry node '%s' to cluster network: %+v", regNode.Name, err)
			}
		}

		// generate the registries.yaml
		regConf, err := RegistryGenerateK3sConfig(ctx, clusterConfig.ClusterCreateOpts.Registries.Use)
		if err != nil {
			return fmt.Errorf("Failed to generate registry config file for k3s: %+v", err)
		}

		// generate the LocalRegistryHosting configmap
		regCm, err := RegistryGenerateLocalRegistryHostingConfigMapYAML(ctx, runtime, clusterConfig.ClusterCreateOpts.Registries.Use)
		if err != nil {
			return fmt.Errorf("Failed to generate LocalRegistryHosting configmap: %+v", err)
		}
		l.Log().Tracef("Writing LocalRegistryHosting YAML:\n%s", string(regCm))
		clusterConfig.ClusterCreateOpts.NodeHooks = append(clusterConfig.ClusterCreateOpts.NodeHooks, k3d.NodeHook{
			Stage: k3d.LifecycleStagePreStart,
			Action: actions.WriteFileAction{
				Runtime:     runtime,
				Content:     regCm,
				Dest:        k3d.DefaultLocalRegistryHostingConfigmapTempPath,
				Mode:        0644,
				Description: "Write LocalRegistryHosting Configmap",
			},
		})

		registryConfig = regConf
	}
	// merge with pre-existing, referenced registries.yaml
	if clusterConfig.ClusterCreateOpts.Registries.Config != nil {
		if registryConfig != nil {
			if err := RegistryMergeConfig(ctx, registryConfig, clusterConfig.ClusterCreateOpts.Registries.Config); err != nil {
				return err
			}
			l.Log().Tracef("Merged registry config: %+v", registryConfig)
		} else {
			registryConfig = clusterConfig.ClusterCreateOpts.Registries.Config
		}
	}
	if registryConfig != nil {
		regConfBytes, err := goyaml.Marshal(&registryConfig)
		if err != nil {
			return fmt.Errorf("Failed to marshal registry configuration: %+v", err)
		}
		clusterConfig.ClusterCreateOpts.NodeHooks = append(clusterConfig.ClusterCreateOpts.NodeHooks, k3d.NodeHook{
			Stage: k3d.LifecycleStagePreStart,
			Action: actions.WriteFileAction{
				Runtime:     runtime,
				Content:     regConfBytes,
				Dest:        k3d.DefaultRegistriesFilePath,
				Mode:        0644,
				Description: "Write Registry Configuration",
			},
		})
	}

	/*
	 * Step 4: Files
	 */

	for id, node := range clusterConfig.Nodes {
		for _, nodefile := range node.Files {
			clusterConfig.Nodes[id].HookActions = append(clusterConfig.Nodes[id].HookActions, k3d.NodeHook{
				Stage: k3d.LifecycleStagePreStart,
				Action: actions.WriteFileAction{
					Runtime:     runtime,
					Content:     nodefile.Content,
					Dest:        nodefile.Destination,
					Mode:        0644,
					Description: nodefile.Description,
				},
			})
		}
	}

	return nil
}

// ClusterPrepNetwork creates a new cluster network, if needed or sets everything up to re-use an existing network
func ClusterPrepNetwork(ctx context.Context, runtime k3drt.Runtime, cluster *k3d.Cluster, clusterCreateOpts *k3d.ClusterCreateOpts) error {
	l.Log().Infoln("Prep: Network")

	// error out if external cluster network should be used but no name was set
	if cluster.Network.Name == "" && cluster.Network.External {
		return fmt.Errorf("Failed to use external network because no name was specified")
	}

	if cluster.Network.Name != "" && cluster.Network.External && cluster.Network.IPAM.IPPrefix.IsValid() {
		return fmt.Errorf("cannot specify subnet for exiting network")
	}

	// generate cluster network name, if not set
	if cluster.Network.Name == "" && !cluster.Network.External {
		cluster.Network.Name = fmt.Sprintf("%s-%s", k3d.DefaultObjectNamePrefix, cluster.Name)
	}

	// handle hostnetwork
	if cluster.Network.Name == "host" {
		if len(cluster.Nodes) > 1 {
			return fmt.Errorf("only one server node supported when using host network")
		}
	}

	// create cluster network or use an existing one
	network, networkExists, err := runtime.CreateNetworkIfNotPresent(ctx, &cluster.Network)
	if err != nil {
		return fmt.Errorf("failed to create cluster network: %w", err)
	}
	cluster.Network = *network
	clusterCreateOpts.GlobalLabels[k3d.LabelNetworkID] = network.ID
	clusterCreateOpts.GlobalLabels[k3d.LabelNetwork] = cluster.Network.Name
	clusterCreateOpts.GlobalLabels[k3d.LabelNetworkIPRange] = cluster.Network.IPAM.IPPrefix.String()
	clusterCreateOpts.GlobalLabels[k3d.LabelNetworkExternal] = strconv.FormatBool(cluster.Network.External)
	if networkExists {
		l.Log().Infof("Re-using existing network '%s' (%s)", network.Name, network.ID)
		clusterCreateOpts.GlobalLabels[k3d.LabelNetworkExternal] = "true" // if the network wasn't created, we say that it's managed externally (important for cluster deletion)
	}

	// just reserve some IPs for k3d (e.g. k3d-tools container), so we don't try to use them again
	if cluster.Network.IPAM.Managed {
		reservedIP, err := GetIP(ctx, runtime, &cluster.Network)
		if err != nil {
			return fmt.Errorf("error reserving IP in new cluster network %s", network.Name)
		}
		cluster.Network.IPAM.IPsUsed = append(cluster.Network.IPAM.IPsUsed, reservedIP)
	}

	return nil
}

func ClusterPrepImageVolume(ctx context.Context, runtime k3drt.Runtime, cluster *k3d.Cluster, clusterCreateOpts *k3d.ClusterCreateOpts) error {
	/*
	 * Cluster-Wide volumes
	 * - image volume (for importing images)
	 */
	imageVolumeName := fmt.Sprintf("%s-%s-images", k3d.DefaultObjectNamePrefix, cluster.Name)
	if err := runtime.CreateVolume(ctx, imageVolumeName, map[string]string{k3d.LabelClusterName: cluster.Name}); err != nil {
		return fmt.Errorf("failed to create image volume '%s' for cluster '%s': %w", imageVolumeName, cluster.Name, err)
	}
	l.Log().Infof("Created image volume %s", imageVolumeName)

	clusterCreateOpts.GlobalLabels[k3d.LabelImageVolume] = imageVolumeName
	cluster.ImageVolume = imageVolumeName
	cluster.Volumes = append(cluster.Volumes, imageVolumeName)

	// attach volume to nodes
	for _, node := range cluster.Nodes {
		node.Volumes = append(node.Volumes, fmt.Sprintf("%s:%s", imageVolumeName, k3d.DefaultImageVolumeMountPath))
	}
	return nil
}

// ClusterCreate creates a new cluster consisting of
// - some containerized k3s nodes
// - a docker network
func ClusterCreate(ctx context.Context, runtime k3drt.Runtime, cluster *k3d.Cluster, clusterCreateOpts *k3d.ClusterCreateOpts) error {
	l.Log().Tracef(`
===== Creating Cluster =====

Runtime:
%+v

Cluster:
%+v

ClusterCreatOpts:
%+v

============================
	`, runtime, cluster, clusterCreateOpts)

	/*
	 * Set up contexts
	 * Used for (early) termination (across API boundaries)
	 */
	clusterCreateCtx := ctx
	if clusterCreateOpts.Timeout > 0*time.Second {
		var cancelClusterCreateCtx context.CancelFunc
		clusterCreateCtx, cancelClusterCreateCtx = context.WithTimeout(ctx, clusterCreateOpts.Timeout)
		defer cancelClusterCreateCtx()
	}

	/*
	 * Docker Machine Special Configuration
	 */
	if cluster.KubeAPI.Host == k3d.DefaultAPIHost && runtime == k3drt.Docker {
		// If the runtime is docker, attempt to use the docker host
		if runtime == k3drt.Docker {
			dockerHost := runtime.GetHost()
			if dockerHost != "" {
				dockerHost = strings.Split(dockerHost, ":")[0] // remove the port
				l.Log().Tracef("Using docker host %s", dockerHost)
				cluster.KubeAPI.Host = dockerHost
			}
		}
	}

	/*
	 * Cluster Token
	 */

	if cluster.Token == "" {
		cluster.Token = GenerateClusterToken()
	}
	clusterCreateOpts.GlobalLabels[k3d.LabelClusterToken] = cluster.Token

	/*
	 * Extra Labels
	 */
	if len(clusterCreateOpts.HostAliases) > 0 {
		hostAliasesJSON, err := json.Marshal(clusterCreateOpts.HostAliases)
		if err != nil {
			return fmt.Errorf("error marshalling hostaliases: %w", err)
		}

		clusterCreateOpts.GlobalLabels[k3d.LabelClusterStartHostAliases] = string(hostAliasesJSON)
	}

	/*
	 * Nodes
	 */

	clusterCreateOpts.GlobalLabels[k3d.LabelClusterName] = cluster.Name
	// Add serverlb url to be used as tls-san value
	// This is used to avoid a fatal error on registering server nodes
	// using loadbalancer
	if clusterCreateOpts.DisableLoadBalancer {
		clusterCreateOpts.GlobalLabels[k3d.LabelServerLoadBalancer] = ""
	} else {
		clusterCreateOpts.GlobalLabels[k3d.LabelServerLoadBalancer] = fmt.Sprintf("%s-%s-serverlb", k3d.DefaultObjectNamePrefix, cluster.Name)
	}

	// agent defaults (per cluster)
	// connection url is always the name of the first server node (index 0) // TODO: change this to the server loadbalancer
	connectionURL := fmt.Sprintf("https://%s:%s", GenerateNodeName(cluster.Name, k3d.ServerRole, 0), k3d.DefaultAPIPort)
	clusterCreateOpts.GlobalLabels[k3d.LabelClusterURL] = connectionURL
	clusterCreateOpts.GlobalEnv = append(clusterCreateOpts.GlobalEnv, fmt.Sprintf("%s=%s", k3s.EnvClusterToken, cluster.Token))

	nodeSetup := func(node *k3d.Node) error {
		// cluster specific settings
		if node.RuntimeLabels == nil {
			node.RuntimeLabels = make(map[string]string) // TODO: maybe create an init function?
		}

		// ensure global labels
		for k, v := range clusterCreateOpts.GlobalLabels {
			node.RuntimeLabels[k] = v
		}

		// ensure global env
		node.Env = append(node.Env, clusterCreateOpts.GlobalEnv...)

		// node role specific settings
		if node.Role == k3d.ServerRole {
			if cluster.Network.IPAM.Managed {
				ip, err := GetIP(ctx, runtime, &cluster.Network)
				if err != nil {
					return fmt.Errorf("failed to find free IP in network %s: %w", cluster.Network.Name, err)
				}
				cluster.Network.IPAM.IPsUsed = append(cluster.Network.IPAM.IPsUsed, ip) // make sure that we're not reusing the same IP next time
				node.IP.Static = true
				node.IP.IP = ip
				node.RuntimeLabels[k3d.LabelNodeStaticIP] = ip.String()
			}

			node.ServerOpts.KubeAPI = cluster.KubeAPI

			// the cluster has an init server node, but its not this one, so connect it to the init node
			if cluster.InitNode != nil && !node.ServerOpts.IsInit {
				node.Env = append(node.Env, fmt.Sprintf("%s=%s", k3s.EnvClusterConnectURL, connectionURL))
				node.RuntimeLabels[k3d.LabelServerIsInit] = "false" // set label, that this server node is not the init server
			}
		} else if node.Role == k3d.AgentRole {
			node.Env = append(node.Env, fmt.Sprintf("%s=%s", k3s.EnvClusterConnectURL, connectionURL))
		}

		node.Networks = []string{cluster.Network.Name}
		node.Restart = true
		node.GPURequest = clusterCreateOpts.GPURequest

		// create node
		l.Log().Infof("Creating node '%s'", node.Name)
		if err := NodeCreate(clusterCreateCtx, runtime, node, k3d.NodeCreateOpts{}); err != nil {
			return fmt.Errorf("failed to create node: %w", err)
		}
		l.Log().Debugf("Created node '%s'", node.Name)
		return nil
	}

	// used for node suffices
	serverCount := 0

	// create init node first
	if cluster.InitNode != nil {
		l.Log().Infoln("Creating initializing server node")
		cluster.InitNode.Args = append(cluster.InitNode.Args, "--cluster-init")
		if cluster.InitNode.RuntimeLabels == nil {
			cluster.InitNode.RuntimeLabels = map[string]string{}
		}
		cluster.InitNode.RuntimeLabels[k3d.LabelServerIsInit] = "true" // set label, that this server node is the init server

		// in case the LoadBalancer was disabled, expose the API Port on the initializing server node
		if clusterCreateOpts.DisableLoadBalancer {
			if cluster.InitNode.Ports == nil {
				cluster.InitNode.Ports = nat.PortMap{}
			}
			cluster.InitNode.Ports[k3d.DefaultAPIPort] = []nat.PortBinding{cluster.KubeAPI.Binding}
		}

		if err := nodeSetup(cluster.InitNode); err != nil {
			return fmt.Errorf("failed init node setup: %w", err)
		}
		serverCount++
	}

	// create all other nodes, but skip the init node
	for _, node := range cluster.Nodes {
		if node.Role == k3d.ServerRole {
			// skip the init node here
			if node == cluster.InitNode {
				continue
			} else if serverCount == 0 && clusterCreateOpts.DisableLoadBalancer {
				// if this is the first server node and the server loadbalancer is disabled, expose the API Port on this server node
				if node.Ports == nil {
					node.Ports = nat.PortMap{}
				}
				node.Ports[k3d.DefaultAPIPort] = []nat.PortBinding{cluster.KubeAPI.Binding}
			}

			time.Sleep(1 * time.Second) // FIXME: arbitrary wait for one second to avoid race conditions of servers registering

			serverCount++
		}
		if node.Role == k3d.ServerRole || node.Role == k3d.AgentRole {
			if err := nodeSetup(node); err != nil {
				return fmt.Errorf("failed setup of server/agent node %s: %w", node.Name, err)
			}
		}
	}

	// WARN, if there are exactly two server nodes: that means we're using etcd, but don't have fault tolerance
	if serverCount == 2 {
		l.Log().Warnln("You're creating 2 server nodes: Please consider creating at least 3 to achieve etcd quorum & fault tolerance")
	}

	/*
	 * Auxiliary Containers
	 */
	// *** ServerLoadBalancer ***
	if !clusterCreateOpts.DisableLoadBalancer {
		if cluster.ServerLoadBalancer == nil {
			l.Log().Infof("No loadbalancer specified, creating a default one...")
			cluster.ServerLoadBalancer = k3d.NewLoadbalancer()
			var err error
			cluster.ServerLoadBalancer.Node, err = LoadbalancerPrepare(ctx, runtime, cluster, &k3d.LoadbalancerCreateOpts{Labels: clusterCreateOpts.GlobalLabels})
			if err != nil {
				return fmt.Errorf("failed to prepare loadbalancer: %w", err)
			}
			cluster.Nodes = append(cluster.Nodes, cluster.ServerLoadBalancer.Node) // append lbNode to list of cluster nodes, so it will be considered during rollback
		}

		if len(cluster.ServerLoadBalancer.Config.Ports) == 0 {
			lbConfig, err := LoadbalancerGenerateConfig(cluster)
			if err != nil {
				return fmt.Errorf("error generating loadbalancer config: %v", err)
			}
			cluster.ServerLoadBalancer.Config = &lbConfig
		}

		// ensure labels
		cluster.ServerLoadBalancer.Node.FillRuntimeLabels()
		for k, v := range clusterCreateOpts.GlobalLabels {
			cluster.ServerLoadBalancer.Node.RuntimeLabels[k] = v
		}

		// prepare to write config to lb container
		configyaml, err := yaml.Marshal(cluster.ServerLoadBalancer.Config)
		if err != nil {
			return fmt.Errorf("failed to marshal loadbalancer config: %w", err)
		}

		writeLbConfigAction := k3d.NodeHook{
			Stage: k3d.LifecycleStagePreStart,
			Action: actions.WriteFileAction{
				Runtime:     runtime,
				Dest:        k3d.DefaultLoadbalancerConfigPath,
				Mode:        0744,
				Content:     configyaml,
				Description: "Write Loadbalancer Configuration",
			},
		}

		cluster.ServerLoadBalancer.Node.HookActions = append(cluster.ServerLoadBalancer.Node.HookActions, writeLbConfigAction)
		cluster.ServerLoadBalancer.Node.Restart = true

		l.Log().Infof("Creating LoadBalancer '%s'", cluster.ServerLoadBalancer.Node.Name)
		if err := NodeCreate(ctx, runtime, cluster.ServerLoadBalancer.Node, k3d.NodeCreateOpts{}); err != nil {
			return fmt.Errorf("error creating loadbalancer: %v", err)
		}
		l.Log().Debugf("Created loadbalancer '%s'", cluster.ServerLoadBalancer.Node.Name)
	}

	return nil
}

// ClusterDelete deletes an existing cluster
func ClusterDelete(ctx context.Context, runtime k3drt.Runtime, cluster *k3d.Cluster, opts k3d.ClusterDeleteOpts) error {
	l.Log().Infof("Deleting cluster '%s'", cluster.Name)
	cluster, err := ClusterGet(ctx, runtime, cluster)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}
	l.Log().Debugf("Cluster Details: %+v", cluster)

	failed := 0
	for _, node := range cluster.Nodes {
		// registry: only delete, if not connected to other networks
		if node.Role == k3d.RegistryRole && !opts.SkipRegistryCheck {
			l.Log().Tracef("Registry Node has %d networks: %+v", len(node.Networks), node)

			// check if node is connected to other networks, that are not
			// - the cluster network
			// - default docker networks
			// -> if so, disconnect it from the cluster network and continue
			connectedToOtherNet := false
			for _, net := range node.Networks {
				if net == cluster.Network.Name {
					continue
				}
				if net == k3d.DefaultRuntimeNetwork || net == "host" {
					continue
				}
				l.Log().Tracef("net: %s", net)
				connectedToOtherNet = true
				break
			}
			if connectedToOtherNet {
				l.Log().Infof("Registry %s is also connected to other (non-default) networks (%+v), not deleting it...", node.Name, node.Networks)
				if err := runtime.DisconnectNodeFromNetwork(ctx, node, cluster.Network.Name); err != nil {
					l.Log().Warnf("Failed to disconnect registry %s from cluster network %s", node.Name, cluster.Network.Name)
				}
				continue
			}
		}

		if err := NodeDelete(ctx, runtime, node, k3d.NodeDeleteOpts{SkipLBUpdate: true}); err != nil {
			l.Log().Warningf("Failed to delete node '%s': Try to delete it manually", node.Name)
			failed++
			continue
		}
	}

	// Delete the cluster network, if it was created for/by this cluster (and if it's not in use anymore)
	if cluster.Network.Name != "" {
		if !cluster.Network.External {
			l.Log().Infof("Deleting cluster network '%s'", cluster.Network.Name)
			if err := runtime.DeleteNetwork(ctx, cluster.Network.Name); err != nil {
				if errors.Is(err, runtimeErr.ErrRuntimeNetworkNotEmpty) { // there are still containers connected to that network
					connectedNodes, err := runtime.GetNodesInNetwork(ctx, cluster.Network.Name) // check, if there are any k3d nodes connected to the cluster
					if err != nil {
						l.Log().Warningf("Failed to check cluster network for connected nodes: %+v", err)
					}

					if len(connectedNodes) > 0 { // there are still k3d-managed containers (aka nodes) connected to the network
						connectedRegistryNodes := util.FilterNodesByRole(connectedNodes, k3d.RegistryRole)
						if len(connectedRegistryNodes) == len(connectedNodes) { // only registry node(s) left in the network
							for _, node := range connectedRegistryNodes {
								l.Log().Debugf("Disconnecting registry node %s from the network...", node.Name)
								if err := runtime.DisconnectNodeFromNetwork(ctx, node, cluster.Network.Name); err != nil {
									l.Log().Warnf("Failed to disconnect registry %s from network %s", node.Name, cluster.Network.Name)
								} else {
									if err := runtime.DeleteNetwork(ctx, cluster.Network.Name); err != nil {
										l.Log().Warningf("Failed to delete cluster network, even after disconnecting registry node(s): %+v", err)
									}
								}
							}
						} else { // besides the registry node(s), there are still other nodes... maybe they still need a registry
							l.Log().Debugf("There are some non-registry nodes left in the network")
						}
					} else {
						l.Log().Warningf("Failed to delete cluster network '%s' because it's still in use: is there another cluster using it?", cluster.Network.Name)
					}
				} else {
					l.Log().Warningf("Failed to delete cluster network '%s': '%+v'", cluster.Network.Name, err)
				}
			}
		} else if cluster.Network.External {
			l.Log().Debugf("Skip deletion of cluster network '%s' because it's managed externally", cluster.Network.Name)
		}
	}

	// delete managed volumes attached to this cluster
	l.Log().Infof("Deleting %d attached volumes...", len(cluster.Volumes))
	for _, vol := range cluster.Volumes {
		l.Log().Debugf("Deleting volume %s...", vol)
		if err := runtime.DeleteVolume(ctx, vol); err != nil {
			l.Log().Warningf("Failed to delete volume '%s' of cluster '%s': %v -> Try to delete it manually", cluster.ImageVolume, cluster.Name, err)
		}
	}

	// return error if we failed to delete a node
	if failed > 0 {
		return fmt.Errorf("Failed to delete %d nodes: Try to delete them manually", failed)
	}
	return nil
}

// ClusterList returns a list of all existing clusters
func ClusterList(ctx context.Context, runtime k3drt.Runtime) ([]*k3d.Cluster, error) {
	l.Log().Traceln("Listing Clusters...")
	nodes, err := runtime.GetNodesByLabel(ctx, k3d.DefaultRuntimeLabels)
	if err != nil {
		return nil, fmt.Errorf("runtime failed to list nodes: %w", err)
	}

	l.Log().Debugf("Found %d nodes", len(nodes))
	if l.Log().GetLevel() == logrus.TraceLevel {
		for _, node := range nodes {
			l.Log().Tracef("Found node %s of role %s", node.Name, node.Role)
		}
	}

	nodes = NodeFilterByRoles(nodes, k3d.ClusterInternalNodeRoles, k3d.ClusterExternalNodeRoles)

	l.Log().Tracef("Found %d cluster-internal nodes", len(nodes))
	if l.Log().GetLevel() == logrus.TraceLevel {
		for _, node := range nodes {
			l.Log().Tracef("Found cluster-internal node %s of role %s belonging to cluster %s", node.Name, node.Role, node.RuntimeLabels[k3d.LabelClusterName])
		}
	}

	clusters := []*k3d.Cluster{}
	// for each node, check, if we can add it to a cluster or add the cluster if it doesn't exist yet
	for _, node := range nodes {
		clusterExists := false
		for _, cluster := range clusters {
			if node.RuntimeLabels[k3d.LabelClusterName] == cluster.Name { // TODO: handle case, where this label doesn't exist
				cluster.Nodes = append(cluster.Nodes, node)
				clusterExists = true
				break
			}
		}
		// cluster is not in the list yet, so we add it with the current node as its first member
		if !clusterExists {
			clusters = append(clusters, &k3d.Cluster{
				Name:  node.RuntimeLabels[k3d.LabelClusterName],
				Nodes: []*k3d.Node{node},
			})
		}
	}

	// enrich cluster structs with label values
	for _, cluster := range clusters {
		if err := populateClusterFieldsFromLabels(cluster); err != nil {
			l.Log().Warnf("Failed to populate cluster fields from node label values for cluster '%s'", cluster.Name)
			l.Log().Warnln(err)
		}
	}
	l.Log().Debugf("Found %d clusters", len(clusters))
	return clusters, nil
}

// populateClusterFieldsFromLabels inspects labels attached to nodes and translates them to struct fields
func populateClusterFieldsFromLabels(cluster *k3d.Cluster) error {
	networkExternalSet := false

	for _, node := range cluster.Nodes {
		// get the name of the cluster network
		if cluster.Network.Name == "" {
			if networkName, ok := node.RuntimeLabels[k3d.LabelNetwork]; ok {
				cluster.Network.Name = networkName
			}
		}

		// check if the network is external
		// since the struct value is a bool, initialized as false, we cannot check if it's unset
		if !cluster.Network.External && !networkExternalSet {
			if networkExternalString, ok := node.RuntimeLabels[k3d.LabelNetworkExternal]; ok {
				if networkExternal, err := strconv.ParseBool(networkExternalString); err == nil {
					cluster.Network.External = networkExternal
					networkExternalSet = true
				}
			}
		}

		// get image volume // TODO: enable external image volumes the same way we do it with networks
		if cluster.ImageVolume == "" {
			if imageVolumeName, ok := node.RuntimeLabels[k3d.LabelImageVolume]; ok {
				cluster.ImageVolume = imageVolumeName
			}
		}

		// get k3s cluster's token
		if cluster.Token == "" {
			if token, ok := node.RuntimeLabels[k3d.LabelClusterToken]; ok {
				cluster.Token = token
			}
		}
	}

	return nil
}

func GetClusterStartOptsFromLabels(cluster *k3d.Cluster) (k3d.ClusterStartOpts, error) {
	clusterStartOpts := k3d.ClusterStartOpts{
		HostAliases: []k3d.HostAlias{},
	}
	for _, node := range cluster.Nodes {
		if len(clusterStartOpts.HostAliases) == 0 {
			if hostAliasesJSON, ok := node.RuntimeLabels[k3d.LabelClusterStartHostAliases]; ok {
				if err := json.Unmarshal([]byte(hostAliasesJSON), &clusterStartOpts.HostAliases); err != nil {
					return clusterStartOpts, fmt.Errorf("error unmarshalling hostaliases JSON from node %s label: %w", node.Name, err)
				}
			}
		}
	}
	return clusterStartOpts, nil
}

var ClusterGetNoNodesFoundError = errors.New("No nodes found for given cluster")

// ClusterGet returns an existing cluster with all fields and node lists populated
func ClusterGet(ctx context.Context, runtime k3drt.Runtime, cluster *k3d.Cluster) (*k3d.Cluster, error) {
	// get nodes that belong to the selected cluster
	nodes, err := runtime.GetNodesByLabel(ctx, map[string]string{k3d.LabelClusterName: cluster.Name})
	if err != nil {
		l.Log().Errorf("Failed to get nodes for cluster '%s': %v", cluster.Name, err)
	}

	if len(nodes) == 0 {
		return nil, ClusterGetNoNodesFoundError
	}

	// append nodes
	for _, node := range nodes {
		// check if there's already a node in the struct
		overwroteExisting := false
		for _, existingNode := range cluster.Nodes {
			// overwrite existing node
			if existingNode.Name == node.Name {
				if err := mergo.Merge(existingNode, node, mergo.WithOverride); err != nil {
					return nil, fmt.Errorf("failed to merge node %s into cluster: %v", node.Name, err)
				}
				overwroteExisting = true
			}
		}

		// no existing node overwritten: append new node
		if !overwroteExisting {
			cluster.Nodes = append(cluster.Nodes, node)
		}
	}

	// Loadbalancer
	if cluster.ServerLoadBalancer == nil {
		for _, node := range cluster.Nodes {
			if node.Role == k3d.LoadBalancerRole {
				cluster.ServerLoadBalancer = &k3d.Loadbalancer{
					Node: node,
				}
			}
		}

		if cluster.ServerLoadBalancer != nil && cluster.ServerLoadBalancer.Node != nil {
			lbcfg, err := GetLoadbalancerConfig(ctx, runtime, cluster)
			if err != nil {
				l.Log().Errorf("error getting loadbalancer config from %s: %v", cluster.ServerLoadBalancer.Node.Name, err)
			}
			cluster.ServerLoadBalancer.Config = &lbcfg
		}
	}

	vols, err := runtime.GetVolumesByLabel(ctx, map[string]string{k3d.LabelClusterName: cluster.Name})
	if err != nil {
		return nil, err
	}
	for _, vol := range vols {
		if !slices.Contains(cluster.Volumes, vol) {
			cluster.Volumes = append(cluster.Volumes, vol)
		}
	}

	if err := populateClusterFieldsFromLabels(cluster); err != nil {
		l.Log().Warnf("Failed to populate cluster fields from node labels: %v", err)
	}

	return cluster, nil
}

// GenerateClusterToken generates a random 20 character string
func GenerateClusterToken() string {
	return util.GenerateRandomString(20)
}

func GenerateNodeName(cluster string, role k3d.Role, suffix int) string {
	return fmt.Sprintf("%s-%s-%s-%d", k3d.DefaultObjectNamePrefix, cluster, role, suffix)
}

// ClusterStart starts a whole cluster (i.e. all nodes of the cluster)
func ClusterStart(ctx context.Context, runtime k3drt.Runtime, cluster *k3d.Cluster, clusterStartOpts k3d.ClusterStartOpts) error {
	l.Log().Infof("Starting cluster '%s'", cluster.Name)

	if clusterStartOpts.Intent == "" {
		clusterStartOpts.Intent = k3d.IntentClusterStart
	}

	if clusterStartOpts.Timeout > 0*time.Second {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, clusterStartOpts.Timeout)
		defer cancel()
	}

	// sort the nodes into categories
	var initNode *k3d.Node
	var servers []*k3d.Node
	var agents []*k3d.Node
	var aux []*k3d.Node
	for _, n := range cluster.Nodes {
		if !n.State.Running {
			if n.Role == k3d.ServerRole {
				if n.ServerOpts.IsInit {
					initNode = n
					continue
				}
				servers = append(servers, n)
			} else if n.Role == k3d.AgentRole {
				agents = append(agents, n)
			} else {
				aux = append(aux, n)
			}
		} else {
			l.Log().Tracef("Node %s already running.", n.Name)
		}
	}

	// sort list of servers for properly ordered sequential start
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})

	/*
	 * Init Node
	 */
	if initNode != nil {
		l.Log().Infoln("Starting the initializing server...")
		if err := NodeStart(ctx, runtime, initNode, &k3d.NodeStartOpts{
			Wait:            true, // always wait for the init node
			NodeHooks:       clusterStartOpts.NodeHooks,
			ReadyLogMessage: k3d.GetReadyLogMessage(initNode, clusterStartOpts.Intent), // initNode means, that we're using etcd -> this will need quorum, so "k3s is up and running" won't happen right now
			EnvironmentInfo: clusterStartOpts.EnvironmentInfo,
		}); err != nil {
			return fmt.Errorf("Failed to start initializing server node: %+v", err)
		}
	}

	/*
	 * Server Nodes
	 */
	if len(servers) > 0 {
		l.Log().Infoln("Starting servers...")
		for _, serverNode := range servers {
			if err := NodeStart(ctx, runtime, serverNode, &k3d.NodeStartOpts{
				Wait:            true,
				NodeHooks:       append(clusterStartOpts.NodeHooks, serverNode.HookActions...),
				EnvironmentInfo: clusterStartOpts.EnvironmentInfo,
			}); err != nil {
				return fmt.Errorf("Failed to start server %s: %+v", serverNode.Name, err)
			}
		}
	} else {
		l.Log().Infoln("All servers already running.")
	}

	/*
	 * Agent Nodes
	 */
	if len(agents) > 0 {
		agentWG, aCtx := errgroup.WithContext(ctx)

		l.Log().Infoln("Starting agents...")
		for _, agentNode := range agents {
			currentAgentNode := agentNode
			agentWG.Go(func() error {
				return NodeStart(aCtx, runtime, currentAgentNode, &k3d.NodeStartOpts{
					Wait:            true,
					NodeHooks:       append(clusterStartOpts.NodeHooks, agentNode.HookActions...),
					EnvironmentInfo: clusterStartOpts.EnvironmentInfo,
				})
			})
		}
		if err := agentWG.Wait(); err != nil {
			return fmt.Errorf("Failed to add one or more agents: %w", err)
		}
	} else {
		l.Log().Infoln("All agents already running.")
	}

	/*
	 * Auxiliary/Helper Nodes
	 */

	if len(aux) > 0 {
		helperWG, hCtx := errgroup.WithContext(ctx)
		l.Log().Infoln("Starting helpers...")
		for _, helperNode := range aux {
			currentHelperNode := helperNode

			helperWG.Go(func() error {
				nodeStartOpts := &k3d.NodeStartOpts{
					NodeHooks:       currentHelperNode.HookActions,
					EnvironmentInfo: clusterStartOpts.EnvironmentInfo,
				}
				if currentHelperNode.Role == k3d.LoadBalancerRole {
					nodeStartOpts.Wait = true
				}

				return NodeStart(hCtx, runtime, currentHelperNode, nodeStartOpts)
			})
		}

		if err := helperWG.Wait(); err != nil {
			return fmt.Errorf("Failed to add one or more helper nodes: %w", err)
		}
	} else {
		l.Log().Infoln("All helpers already running.")
	}

	/*
	 * Additional Cluster Preparation (post start)
	 */

	if len(servers) > 0 || len(agents) > 0 { // TODO: make checks for required cluster start actions cleaner
		postStartErrgrp, postStartErrgrpCtx := errgroup.WithContext(ctx)

		/*** DNS ***/

		// -> skip if hostnetwork mode
		if cluster.Network.Name == "host" {
			l.Log().Debugf("Not injecting hostAliases into /etc/hosts and CoreDNS as clusternetwork is 'host'")
		} else {
			// -> add hostAliases to /etc/hosts in all nodes
			// --> inject host-gateway as host.k3d.internal
			clusterStartOpts.HostAliases = append(clusterStartOpts.HostAliases, k3d.HostAlias{
				IP:        clusterStartOpts.EnvironmentInfo.HostGateway.String(),
				Hostnames: []string{"host.k3d.internal"},
			})

			for _, node := range append(servers, agents...) {
				currNode := node
				postStartErrgrp.Go(func() error {
					return NewHostAliasesInjectEtcHostsAction(runtime, clusterStartOpts.HostAliases).Run(postStartErrgrpCtx, currNode)
				})
			}

			// -> inject hostAliases and network members into CoreDNS configmap
			if len(servers) > 0 {
				postStartErrgrp.Go(func() error {
					hosts := ""

					// hosts: hostAliases (including host.k3d.internal)
					for _, hostAlias := range clusterStartOpts.HostAliases {
						hosts += fmt.Sprintf("%s %s\n", hostAlias.IP, strings.Join(hostAlias.Hostnames, " "))
					}

					// more hosts: network members ("neighbor" containers)
					net, err := runtime.GetNetwork(postStartErrgrpCtx, &cluster.Network)
					if err != nil {
						return fmt.Errorf("failed to get cluster network %s to inject host records into CoreDNS: %w", cluster.Network.Name, err)
					}
					for _, member := range net.Members {
						hosts += fmt.Sprintf("%s %s\n", member.IP.String(), member.Name)
					}

					// inject CoreDNS configmap
					l.Log().Infof("Injecting records for hostAliases (incl. host.k3d.internal) and for %d network members into CoreDNS configmap...", len(net.Members))
					act := actions.RewriteFileAction{
						Runtime: runtime,
						Path:    "/var/lib/rancher/k3s/server/manifests/coredns.yaml",
						Mode:    0744,
						RewriteFunc: func(input []byte) ([]byte, error) {
							split, err := util.SplitYAML(input)
							if err != nil {
								return nil, fmt.Errorf("error splitting yaml: %w", err)
							}

							var outputBuf bytes.Buffer
							outputEncoder := util.NewYAMLEncoder(&outputBuf)

							for _, d := range split {
								var doc map[string]interface{}
								if err := yaml.Unmarshal(d, &doc); err != nil {
									return nil, err
								}
								if kind, ok := doc["kind"]; ok {
									if strings.ToLower(kind.(string)) == "configmap" {
										configmapData, ok := doc["data"].(map[string]interface{})
										if !ok {
											return nil, fmt.Errorf("invalid ConfigMap data type: %T", doc["data"])
										}
										configmapData["NodeHosts"] = hosts
									}
								}
								if err := outputEncoder.Encode(doc); err != nil {
									return nil, err
								}
							}
							_ = outputEncoder.Close()
							return outputBuf.Bytes(), nil
						},
					}

					// get the first server in the list and run action on it once it's ready for it
					for _, n := range servers {
						// do not try to run the action, if CoreDNS is disabled on K3s level
						for _, flag := range n.Args {
							if strings.HasPrefix(flag, "--disable") && strings.Contains(flag, "coredns") {
								l.Log().Debugf("CoreDNS disabled in K3s via flag `%s`. Not trying to use it.", flag)
								return nil
							}
						}
						ts, err := time.Parse("2006-01-02T15:04:05.999999999Z", n.State.Started)
						if err != nil {
							return err
						}
						if err := NodeWaitForLogMessage(postStartErrgrpCtx, runtime, n, "Cluster dns configmap", ts.Truncate(time.Second)); err != nil {
							return err
						}
						return act.Run(postStartErrgrpCtx, n) // nolint:staticcheck // FIXME: Does this loop really only concern the first server? (SA4004: the surrounding loop is unconditionally terminated (staticcheck))
					}
					return nil
				})
			}
		}

		if err := postStartErrgrp.Wait(); err != nil {
			return fmt.Errorf("error during post-start cluster preparation: %w", err)
		}
	}

	return nil
}

// ClusterStop stops a whole cluster (i.e. all nodes of the cluster)
func ClusterStop(ctx context.Context, runtime k3drt.Runtime, cluster *k3d.Cluster) error {
	l.Log().Infof("Stopping cluster '%s'", cluster.Name)

	failed := 0
	for _, node := range cluster.Nodes {
		if err := runtime.StopNode(ctx, node); err != nil {
			l.Log().Warningf("Failed to stop node '%s': Try to stop it manually", node.Name)
			failed++
			continue
		}
	}

	if failed > 0 {
		return fmt.Errorf("Failed to stop %d nodes: Try to stop them manually", failed)
	}

	l.Log().Infof("Stopped cluster '%s'", cluster.Name)
	return nil
}

// SortClusters : in place sort cluster list by cluster name alphabetical order
func SortClusters(clusters []*k3d.Cluster) []*k3d.Cluster {
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Name < clusters[j].Name
	})
	return clusters
}

func prepCreateLocalRegistryHostingConfigMap(ctx context.Context, runtime k3drt.Runtime, cluster *k3d.Cluster) error {
	success := false
	for _, node := range cluster.Nodes {
		if node.Role == k3d.AgentRole || node.Role == k3d.ServerRole {
			err := runtime.ExecInNode(ctx, node, []string{"sh", "-c", fmt.Sprintf("kubectl apply -f %s", k3d.DefaultLocalRegistryHostingConfigmapTempPath)})
			if err == nil {
				success = true
				break
			} else {
				l.Log().Debugf("Failed to create LocalRegistryHosting ConfigMap in node %s: %+v", node.Name, err)
			}
		}
	}
	if !success {
		l.Log().Warnf("Failed to create LocalRegistryHosting ConfigMap")
	}
	return nil
}

// ClusterEditChangesetSimple modifies an existing cluster with a given SimpleConfig changeset
func ClusterEditChangesetSimple(ctx context.Context, runtime k3drt.Runtime, cluster *k3d.Cluster, changeset *config.SimpleConfig) error {
	// nodeCount := len(cluster.Nodes)
	nodeList := cluster.Nodes

	// === Ports ===

	existingLB := cluster.ServerLoadBalancer
	lbChangeset := &k3d.Loadbalancer{}

	// copy existing loadbalancer
	lbChangesetNode, err := CopyNode(ctx, existingLB.Node, CopyNodeOpts{keepState: false})
	if err != nil {
		return fmt.Errorf("error copying existing loadbalancer: %w", err)
	}

	lbChangeset.Node = lbChangesetNode

	// copy config from existing loadbalancer
	lbChangesetConfig, err := copystruct.Copy(existingLB.Config)
	if err != nil {
		return fmt.Errorf("error copying config from existing loadbalancer: %w", err)
	}

	lbChangeset.Config = lbChangesetConfig.(*k3d.LoadbalancerConfig)

	// loop over ports
	if len(changeset.Ports) > 0 {
		// 1. ensure that there are only supported suffices in the node filters // TODO: overly complex right now, needs simplification
		for _, portWithNodeFilters := range changeset.Ports {
			filteredNodes, err := util.FilterNodesWithSuffix(nodeList, portWithNodeFilters.NodeFilters)
			if err != nil {
				return fmt.Errorf("failed to filter nodes: %w", err)
			}

			for suffix := range filteredNodes {
				switch suffix {
				case "proxy", util.NodeFilterSuffixNone, util.NodeFilterMapKeyAll:
					continue
				default:
					return fmt.Errorf("error: 'cluster edit' does not (yet) support the '%s' opt/suffix for adding ports", suffix)
				}
			}
		}

		// 2. transform
		cluster.ServerLoadBalancer = lbChangeset // we're working with pointers, so let's point to the changeset here to not update the original that we keep as a reference
		if err := TransformPorts(ctx, runtime, cluster, changeset.Ports); err != nil {
			return fmt.Errorf("error transforming port config %s: %w", changeset.Ports, err)
		}
	}

	l.Log().Debugf("ORIGINAL:\n> Ports: %+v\n> Config: %+v\nCHANGESET:\n> Ports: %+v\n> Config: %+v", existingLB.Node.Ports, existingLB.Config, lbChangeset.Node.Ports, lbChangeset.Config)

	// prepare to write config to lb container
	configyaml, err := yaml.Marshal(lbChangeset.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal loadbalancer config changeset: %w", err)
	}
	writeLbConfigAction := k3d.NodeHook{
		Stage: k3d.LifecycleStagePreStart,
		Action: actions.WriteFileAction{
			Runtime:     runtime,
			Dest:        k3d.DefaultLoadbalancerConfigPath,
			Mode:        0744,
			Content:     configyaml,
			Description: "Write Loadbalancer Configuration",
		},
	}
	if lbChangeset.Node.HookActions == nil {
		lbChangeset.Node.HookActions = []k3d.NodeHook{}
	}
	lbChangeset.Node.HookActions = append(lbChangeset.Node.HookActions, writeLbConfigAction)

	if err := NodeReplace(ctx, runtime, existingLB.Node, lbChangeset.Node); err != nil {
		return fmt.Errorf("error replacing loadbalancer node: %w", err)
	}

	return nil
}
