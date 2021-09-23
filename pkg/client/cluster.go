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
package client

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io/ioutil"
	"sort"
	"strconv"
	"time"

	gort "runtime"

	"github.com/docker/go-connections/nat"
	"github.com/imdario/mergo"
	copystruct "github.com/mitchellh/copystructure"
	"github.com/rancher/k3d/v5/pkg/actions"
	config "github.com/rancher/k3d/v5/pkg/config/v1alpha3"
	l "github.com/rancher/k3d/v5/pkg/logger"
	k3drt "github.com/rancher/k3d/v5/pkg/runtimes"
	"github.com/rancher/k3d/v5/pkg/runtimes/docker"
	runtimeErr "github.com/rancher/k3d/v5/pkg/runtimes/errors"
	"github.com/rancher/k3d/v5/pkg/types"
	k3d "github.com/rancher/k3d/v5/pkg/types"
	"github.com/rancher/k3d/v5/pkg/types/k3s"
	"github.com/rancher/k3d/v5/pkg/util"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"
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
	go EnsureToolsNode(ctx, runtime, &clusterConfig.Cluster)

	/*
	 * Step 1: Create Containers
	 */
	if err := ClusterCreate(ctx, runtime, &clusterConfig.Cluster, &clusterConfig.ClusterCreateOpts); err != nil {
		return fmt.Errorf("Failed Cluster Creation: %+v", err)
	}

	/*
	 * Step 2: Pre-Start Configuration
	 */
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

	var registryConfig *k3s.Registry

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
				Runtime: runtime,
				Content: regCm,
				Dest:    k3d.DefaultLocalRegistryHostingConfigmapTempPath,
				Mode:    0644,
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
		regConfBytes, err := yaml.Marshal(&registryConfig)
		if err != nil {
			return fmt.Errorf("Failed to marshal registry configuration: %+v", err)
		}
		clusterConfig.ClusterCreateOpts.NodeHooks = append(clusterConfig.ClusterCreateOpts.NodeHooks, k3d.NodeHook{
			Stage: k3d.LifecycleStagePreStart,
			Action: actions.WriteFileAction{
				Runtime: runtime,
				Content: regConfBytes,
				Dest:    k3d.DefaultRegistriesFilePath,
				Mode:    0644,
			},
		})
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

	if cluster.Network.Name != "" && cluster.Network.External && !cluster.Network.IPAM.IPPrefix.IsZero() {
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

	clusterCreateOpts.GlobalLabels[k3d.LabelImageVolume] = imageVolumeName
	cluster.ImageVolume = imageVolumeName

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
		if gort.GOOS == "windows" || gort.GOOS == "darwin" {
			l.Log().Tracef("Running on %s: checking if it's using docker-machine", gort.GOOS)
			machineIP, err := runtime.(docker.Docker).GetDockerMachineIP()
			if err != nil {
				l.Log().Warnf("Using docker-machine, but failed to get it's IP: %+v", err)
			} else if machineIP != "" {
				l.Log().Infof("Using the docker-machine IP %s to connect to the Kubernetes API", machineIP)
				cluster.KubeAPI.Host = machineIP
				cluster.KubeAPI.Binding.HostIP = machineIP
			} else {
				l.Log().Traceln("Not using docker-machine")
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
	 * Nodes
	 */

	clusterCreateOpts.GlobalLabels[k3d.LabelClusterName] = cluster.Name

	// agent defaults (per cluster)
	// connection url is always the name of the first server node (index 0) // TODO: change this to the server loadbalancer
	connectionURL := fmt.Sprintf("https://%s:%s", GenerateNodeName(cluster.Name, k3d.ServerRole, 0), k3d.DefaultAPIPort)
	clusterCreateOpts.GlobalLabels[k3d.LabelClusterURL] = connectionURL
	clusterCreateOpts.GlobalEnv = append(clusterCreateOpts.GlobalEnv, fmt.Sprintf("%s=%s", k3d.K3sEnvClusterToken, cluster.Token))

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
				node.Env = append(node.Env, fmt.Sprintf("%s=%s", k3d.K3sEnvClusterConnectURL, connectionURL))
				node.RuntimeLabels[k3d.LabelServerIsInit] = "false" // set label, that this server node is not the init server
			}

		} else if node.Role == k3d.AgentRole {
			node.Env = append(node.Env, fmt.Sprintf("%s=%s", k3d.K3sEnvClusterConnectURL, connectionURL))
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

		// start node
		//return NodeStart(clusterCreateCtx, runtime, node, k3d.NodeStartOpts{PreStartActions: clusterCreateOpts.NodeHookActions})
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
			lbNode, err := LoadbalancerPrepare(ctx, runtime, cluster, &k3d.LoadbalancerCreateOpts{Labels: clusterCreateOpts.GlobalLabels})
			if err != nil {
				return fmt.Errorf("failed to prepare loadbalancer: %w", err)
			}
			cluster.Nodes = append(cluster.Nodes, lbNode) // append lbNode to list of cluster nodes, so it will be considered during rollback
		}

		if len(cluster.ServerLoadBalancer.Config.Ports) == 0 {
			lbConfig, err := LoadbalancerGenerateConfig(cluster)
			if err != nil {
				return fmt.Errorf("error generating loadbalancer config: %v", err)
			}
			cluster.ServerLoadBalancer.Config = &lbConfig
		}

		cluster.ServerLoadBalancer.Node.RuntimeLabels = clusterCreateOpts.GlobalLabels

		// prepare to write config to lb container
		configyaml, err := yaml.Marshal(cluster.ServerLoadBalancer.Config)
		if err != nil {
			return fmt.Errorf("failed to marshal loadbalancer config: %w", err)
		}

		writeLbConfigAction := k3d.NodeHook{
			Stage: k3d.LifecycleStagePreStart,
			Action: actions.WriteFileAction{
				Runtime: runtime,
				Dest:    k3d.DefaultLoadbalancerConfigPath,
				Mode:    0744,
				Content: configyaml,
			},
		}

		cluster.ServerLoadBalancer.Node.HookActions = append(cluster.ServerLoadBalancer.Node.HookActions, writeLbConfigAction)

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
				if net == "bridge" || net == "host" {
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

	// delete image volume
	if cluster.ImageVolume != "" {
		l.Log().Infof("Deleting image volume '%s'", cluster.ImageVolume)
		if err := runtime.DeleteVolume(ctx, cluster.ImageVolume); err != nil {
			l.Log().Warningf("Failed to delete image volume '%s' of cluster '%s': Try to delete it manually", cluster.ImageVolume, cluster.Name)
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
				mergo.MergeWithOverwrite(existingNode, node)
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
func ClusterStart(ctx context.Context, runtime k3drt.Runtime, cluster *k3d.Cluster, clusterStartOpts types.ClusterStartOpts) error {
	l.Log().Infof("Starting cluster '%s'", cluster.Name)

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
	}

	// TODO: remove trace logs below
	l.Log().Traceln("Servers before sort:")
	for i, n := range servers {
		l.Log().Tracef("Server %d - %s", i, n.Name)
	}
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})
	l.Log().Traceln("Servers after sort:")
	for i, n := range servers {
		l.Log().Tracef("Server %d - %s", i, n.Name)
	}

	/*
	 * Init Node
	 */
	if initNode != nil {
		l.Log().Infoln("Starting the initializing server...")
		if err := NodeStart(ctx, runtime, initNode, &k3d.NodeStartOpts{
			Wait:            true, // always wait for the init node
			NodeHooks:       clusterStartOpts.NodeHooks,
			ReadyLogMessage: "Running kube-apiserver", // initNode means, that we're using etcd -> this will need quorum, so "k3s is up and running" won't happen right now
			EnvironmentInfo: clusterStartOpts.EnvironmentInfo,
		}); err != nil {
			return fmt.Errorf("Failed to start initializing server node: %+v", err)
		}
	}

	/*
	 * Server Nodes
	 */
	l.Log().Infoln("Starting servers...")
	for _, serverNode := range servers {
		if err := NodeStart(ctx, runtime, serverNode, &k3d.NodeStartOpts{
			Wait:            true,
			NodeHooks:       clusterStartOpts.NodeHooks,
			EnvironmentInfo: clusterStartOpts.EnvironmentInfo,
		}); err != nil {
			return fmt.Errorf("Failed to start server %s: %+v", serverNode.Name, err)
		}
	}

	/*
	 * Agent Nodes
	 */

	agentWG, aCtx := errgroup.WithContext(ctx)

	l.Log().Infoln("Starting agents...")
	for _, agentNode := range agents {
		currentAgentNode := agentNode
		agentWG.Go(func() error {
			return NodeStart(aCtx, runtime, currentAgentNode, &k3d.NodeStartOpts{
				Wait:            true,
				NodeHooks:       clusterStartOpts.NodeHooks,
				EnvironmentInfo: clusterStartOpts.EnvironmentInfo,
			})
		})
	}
	if err := agentWG.Wait(); err != nil {
		return fmt.Errorf("Failed to add one or more agents: %w", err)
	}

	/*
	 * Auxiliary/Helper Nodes
	 */

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

	/*
	 * Additional Cluster Preparation
	 */

	/*** DNS ***/

	// add /etc/hosts and CoreDNS entry for host.k3d.internal, referring to the host system
	if err := prepInjectHostIP(ctx, runtime, cluster, &clusterStartOpts); err != nil {
		return fmt.Errorf("failed to inject host IP: %w", err)
	}

	// create host records in CoreDNS for external registries
	if err := prepCoreDNSInjectNetworkMembers(ctx, runtime, cluster); err != nil {
		return fmt.Errorf("failed to patch CoreDNS with network members: %w", err)
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

// corednsAddHost adds a host entry to the CoreDNS configmap if it doesn't exist (a host entry is a single line of the form "IP HOST")
func corednsAddHost(ctx context.Context, runtime k3drt.Runtime, cluster *k3d.Cluster, ip string, name string) error {
	hostsEntry := fmt.Sprintf("%s %s", ip, name)
	patchCmd := `patch=$(kubectl get cm coredns -n kube-system --template='{{.data.NodeHosts}}' | sed -n -E -e '/[0-9\.]{4,12}\s` + name + `$/!p' -e '$a` + hostsEntry + `' | tr '\n' '^' | busybox xargs -0 printf '{"data": {"NodeHosts":"%s"}}'| sed -E 's%\^%\\n%g') && kubectl patch cm coredns -n kube-system -p="$patch"`
	successInjectCoreDNSEntry := false
	for _, node := range cluster.Nodes {
		if node.Role == k3d.AgentRole || node.Role == k3d.ServerRole {
			logreader, err := runtime.ExecInNodeGetLogs(ctx, node, []string{"sh", "-c", patchCmd})
			if err == nil {
				successInjectCoreDNSEntry = true
				break
			} else {
				msg := fmt.Sprintf("error patching the CoreDNS ConfigMap to include entry '%s': %+v", hostsEntry, err)
				readlogs, err := ioutil.ReadAll(logreader)
				if err != nil {
					l.Log().Debugf("error reading the logs from failed CoreDNS patch exec process in node %s: %v", node.Name, err)
				} else {
					msg += fmt.Sprintf("\nLogs: %s", string(readlogs))
				}
				l.Log().Debugln(msg)
			}
		}
	}
	if !successInjectCoreDNSEntry {
		return fmt.Errorf("Failed to patch CoreDNS ConfigMap to include entry '%s' (see debug logs)", hostsEntry)
	}
	return nil
}

// prepInjectHostIP adds /etc/hosts and CoreDNS entry for host.k3d.internal, referring to the host system
func prepInjectHostIP(ctx context.Context, runtime k3drt.Runtime, cluster *k3d.Cluster, clusterStartOpts *k3d.ClusterStartOpts) error {
	if cluster.Network.Name == "host" {
		l.Log().Tracef("Not injecting hostIP as clusternetwork is 'host'")
		return nil
	}
	l.Log().Infoln("Trying to get IP of the docker host and inject it into the cluster as 'host.k3d.internal' for easy access")
	hostIP := clusterStartOpts.EnvironmentInfo.HostGateway
	hostsEntry := fmt.Sprintf("%s %s", hostIP.String(), k3d.DefaultK3dInternalHostRecord)
	l.Log().Debugf("Adding extra host entry '%s'...", hostsEntry)
	for _, node := range cluster.Nodes {
		if err := runtime.ExecInNode(ctx, node, []string{"sh", "-c", fmt.Sprintf("echo '%s' >> /etc/hosts", hostsEntry)}); err != nil {
			return fmt.Errorf("failed to add extra entry '%s' to /etc/hosts in node '%s': %w", hostsEntry, node.Name, err)
		}
	}
	l.Log().Debugf("Successfully added host record \"%s\" to /etc/hosts in all nodes", hostsEntry)

	err := corednsAddHost(ctx, runtime, cluster, hostIP.String(), k3d.DefaultK3dInternalHostRecord)
	if err != nil {
		return fmt.Errorf("failed to inject host record \"%s\" into CoreDNS ConfigMap: %w", hostsEntry, err)
	}
	l.Log().Debugf("Successfully added host record \"%s\" to the CoreDNS ConfigMap ", hostsEntry)
	return nil
}

func prepCoreDNSInjectNetworkMembers(ctx context.Context, runtime k3drt.Runtime, cluster *k3d.Cluster) error {
	net, err := runtime.GetNetwork(ctx, &cluster.Network)
	if err != nil {
		return fmt.Errorf("failed to get cluster network %s to inject host records into CoreDNS: %v", cluster.Network.Name, err)
	}
	l.Log().Debugf("Adding %d network members to coredns", len(net.Members))
	for _, member := range net.Members {
		hostsEntry := fmt.Sprintf("%s %s", member.IP.String(), member.Name)
		if err := corednsAddHost(ctx, runtime, cluster, member.IP.String(), member.Name); err != nil {
			return fmt.Errorf("failed to add host entry \"%s\" into CoreDNS: %v", hostsEntry, err)
		}
	}
	return nil
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
	if success == false {
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
			Runtime: runtime,
			Dest:    k3d.DefaultLoadbalancerConfigPath,
			Mode:    0744,
			Content: configyaml,
		},
	}
	if lbChangeset.Node.HookActions == nil {
		lbChangeset.Node.HookActions = []k3d.NodeHook{}
	}
	lbChangeset.Node.HookActions = append(lbChangeset.Node.HookActions, writeLbConfigAction)

	NodeReplace(ctx, runtime, existingLB.Node, lbChangeset.Node)

	return nil
}
