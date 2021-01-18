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

package config

import (
	"context"
	"fmt"

	"github.com/docker/go-connections/nat"
	cliutil "github.com/rancher/k3d/v4/cmd/util" // TODO: move parseapiport to pkg
	conf "github.com/rancher/k3d/v4/pkg/config/v1alpha1"
	"github.com/rancher/k3d/v4/pkg/runtimes"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	"github.com/rancher/k3d/v4/pkg/types/k3s"
	"github.com/rancher/k3d/v4/pkg/util"
	"github.com/rancher/k3d/v4/version"
	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"
)

// TransformSimpleToClusterConfig transforms a simple configuration to a full-fledged cluster configuration
func TransformSimpleToClusterConfig(ctx context.Context, runtime runtimes.Runtime, simpleConfig conf.SimpleConfig) (*conf.ClusterConfig, error) {

	// set default cluster name
	if simpleConfig.Name == "" {
		simpleConfig.Name = k3d.DefaultClusterName
	}

	// fetch latest image
	if simpleConfig.Image == "latest" {
		simpleConfig.Image = version.GetK3sVersion(true)
	}

	clusterNetwork := k3d.ClusterNetwork{}
	if simpleConfig.Network != "" {
		clusterNetwork.Name = simpleConfig.Network
		clusterNetwork.External = true
	}

	// -> API
	if simpleConfig.ExposeAPI.Host == "" {
		simpleConfig.ExposeAPI.Host = k3d.DefaultAPIHost
	}
	if simpleConfig.ExposeAPI.HostIP == "" {
		simpleConfig.ExposeAPI.HostIP = k3d.DefaultAPIHost
	}

	kubeAPIExposureOpts := &k3d.ExposureOpts{
		Host: simpleConfig.ExposeAPI.Host,
	}
	kubeAPIExposureOpts.Port = k3d.DefaultAPIPort
	kubeAPIExposureOpts.Binding = nat.PortBinding{
		HostIP:   simpleConfig.ExposeAPI.HostIP,
		HostPort: simpleConfig.ExposeAPI.HostPort,
	}

	// FILL CLUSTER CONFIG
	newCluster := k3d.Cluster{
		Name:    simpleConfig.Name,
		Network: clusterNetwork,
		Token:   simpleConfig.ClusterToken,
		KubeAPI: kubeAPIExposureOpts,
	}

	// -> NODES
	newCluster.Nodes = []*k3d.Node{}

	if !simpleConfig.Options.K3dOptions.DisableLoadbalancer {
		newCluster.ServerLoadBalancer = &k3d.Node{
			Role: k3d.LoadBalancerRole,
		}
	}

	/*************
	 * Add Nodes *
	 *************/

	for i := 0; i < simpleConfig.Servers; i++ {
		serverNode := k3d.Node{
			Role:       k3d.ServerRole,
			Image:      simpleConfig.Image,
			Args:       simpleConfig.Options.K3sOptions.ExtraServerArgs,
			ServerOpts: k3d.ServerOpts{},
		}

		// first server node will be init node if we have more than one server specified but no external datastore
		if i == 0 && simpleConfig.Servers > 1 {
			serverNode.ServerOpts.IsInit = true
			newCluster.InitNode = &serverNode
		}

		newCluster.Nodes = append(newCluster.Nodes, &serverNode)
	}

	for i := 0; i < simpleConfig.Agents; i++ {
		agentNode := k3d.Node{
			Role:  k3d.AgentRole,
			Image: simpleConfig.Image,
			Args:  simpleConfig.Options.K3sOptions.ExtraAgentArgs,
		}
		newCluster.Nodes = append(newCluster.Nodes, &agentNode)
	}

	/****************************
	 * Extra Node Configuration *
	 ****************************/

	// -> VOLUMES
	nodeCount := simpleConfig.Servers + simpleConfig.Agents
	nodeList := newCluster.Nodes
	if !simpleConfig.Options.K3dOptions.DisableLoadbalancer {
		nodeCount++
		nodeList = append(nodeList, newCluster.ServerLoadBalancer)
	}
	for _, volumeWithNodeFilters := range simpleConfig.Volumes {
		nodes, err := util.FilterNodes(newCluster.Nodes, volumeWithNodeFilters.NodeFilters)
		if err != nil {
			return nil, err
		}

		for _, node := range nodes {
			node.Volumes = append(node.Volumes, volumeWithNodeFilters.Volume)
		}
	}

	// -> PORTS
	for _, portWithNodeFilters := range simpleConfig.Ports {
		if len(portWithNodeFilters.NodeFilters) == 0 && nodeCount > 1 {
			return nil, fmt.Errorf("Portmapping '%s' lacks a node filter, but there's more than one node", portWithNodeFilters.Port)
		}

		nodes, err := util.FilterNodes(nodeList, portWithNodeFilters.NodeFilters)
		if err != nil {
			return nil, err
		}

		for _, node := range nodes {
			portmappings, err := nat.ParsePortSpec(portWithNodeFilters.Port)
			if err != nil {
				return nil, fmt.Errorf("Failed to parse port spec '%s': %+v", portWithNodeFilters.Port, err)
			}
			if node.Ports == nil {
				node.Ports = nat.PortMap{}
			}
			for _, pm := range portmappings {
				if _, exists := node.Ports[pm.Port]; exists {
					node.Ports[pm.Port] = append(node.Ports[pm.Port], pm.Binding)
				} else {
					node.Ports[pm.Port] = []nat.PortBinding{pm.Binding}
				}
			}
		}
	}

	// -> LABELS
	for _, labelWithNodeFilters := range simpleConfig.Labels {
		if len(labelWithNodeFilters.NodeFilters) == 0 && nodeCount > 1 {
			return nil, fmt.Errorf("Labelmapping '%s' lacks a node filter, but there's more than one node", labelWithNodeFilters.Label)
		}

		nodes, err := util.FilterNodes(nodeList, labelWithNodeFilters.NodeFilters)
		if err != nil {
			return nil, err
		}

		for _, node := range nodes {
			if node.Labels == nil {
				node.Labels = make(map[string]string) // ensure that the map is initialized
			}
			k, v := util.SplitLabelKeyValue(labelWithNodeFilters.Label)
			node.Labels[k] = v
		}
	}

	// -> ENV
	for _, envVarWithNodeFilters := range simpleConfig.Env {
		if len(envVarWithNodeFilters.NodeFilters) == 0 && nodeCount > 1 {
			return nil, fmt.Errorf("EnvVarMapping '%s' lacks a node filter, but there's more than one node", envVarWithNodeFilters.EnvVar)
		}

		nodes, err := util.FilterNodes(nodeList, envVarWithNodeFilters.NodeFilters)
		if err != nil {
			return nil, err
		}

		for _, node := range nodes {
			node.Env = append(node.Env, envVarWithNodeFilters.EnvVar)
		}
	}

	/**************************
	 * Cluster Create Options *
	 **************************/

	clusterCreateOpts := k3d.ClusterCreateOpts{
		DisableImageVolume:  simpleConfig.Options.K3dOptions.DisableImageVolume,
		WaitForServer:       simpleConfig.Options.K3dOptions.Wait,
		Timeout:             simpleConfig.Options.K3dOptions.Timeout,
		DisableLoadBalancer: simpleConfig.Options.K3dOptions.DisableLoadbalancer,
		K3sServerArgs:       simpleConfig.Options.K3sOptions.ExtraServerArgs,
		K3sAgentArgs:        simpleConfig.Options.K3sOptions.ExtraAgentArgs,
		GlobalLabels:        map[string]string{}, // empty init
		GlobalEnv:           []string{},          // empty init
	}

	// ensure, that we have the default object labels
	for k, v := range k3d.DefaultObjectLabels {
		clusterCreateOpts.GlobalLabels[k] = v
	}

	/*
	 * Registries
	 */
	if simpleConfig.Registries.Create {
		regPort, err := cliutil.ParsePortExposureSpec("random", k3d.DefaultRegistryPort)
		if err != nil {
			return nil, fmt.Errorf("Failed to get port for registry: %+v", err)
		}
		clusterCreateOpts.Registries.Create = &k3d.Registry{
			Host:         fmt.Sprintf("%s-%s-registry", k3d.DefaultObjectNamePrefix, newCluster.Name),
			Image:        fmt.Sprintf("%s:%s", k3d.DefaultRegistryImageRepo, k3d.DefaultRegistryImageTag),
			ExposureOpts: *regPort,
		}
	}

	for _, usereg := range simpleConfig.Registries.Use {
		reg, err := util.ParseRegistryRef(usereg)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse use-registry string  '%s': %+v", usereg, err)
		}
		log.Tracef("Parsed registry reference: %+v", reg)
		clusterCreateOpts.Registries.Use = append(clusterCreateOpts.Registries.Use, reg)
	}
	var k3sRegistry *k3s.Registry
	if err := yaml.Unmarshal([]byte(simpleConfig.Registries.Config), k3sRegistry); err != nil {
		return nil, fmt.Errorf("Failed to read registry configuration: %+v", err)
	}
	log.Tracef("Registry: read config from input:\n%+v", k3sRegistry)

	/**********************
	 * Kubeconfig Options *
	 **********************/

	// Currently, the kubeconfig options for the cluster config are the same as for the simple config

	/******************************
	 * Create Full Cluster Config *
	 ******************************/

	clusterConfig := &conf.ClusterConfig{
		Cluster:           newCluster,
		ClusterCreateOpts: clusterCreateOpts,
		KubeconfigOpts:    simpleConfig.Options.KubeconfigOptions,
	}

	return clusterConfig, nil
}
