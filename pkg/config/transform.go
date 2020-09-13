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

	"github.com/rancher/k3d/v3/cmd/util"
	"github.com/rancher/k3d/v3/pkg/runtimes"
	k3d "github.com/rancher/k3d/v3/pkg/types"
)

// TransformSimpleToClusterConfig transforms a simple configuration to a full-fledged cluster configuration
func TransformSimpleToClusterConfig(ctx context.Context, runtime runtimes.Runtime, simpleConfig SimpleConfig) (*ClusterConfig, error) {

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

	// FILL CLUSTER CONFIG
	newCluster := k3d.Cluster{
		Name:    simpleConfig.Name,
		Network: clusterNetwork,
		Token:   simpleConfig.ClusterToken,
		ClusterCreateOpts: &k3d.ClusterCreateOpts{
			DisableImageVolume:  simpleConfig.Options.K3dOptions.DisableImageVolume,
			WaitForServer:       simpleConfig.Options.K3dOptions.Wait,
			Timeout:             simpleConfig.Options.K3dOptions.Timeout,
			DisableLoadBalancer: simpleConfig.Options.K3dOptions.DisableLoadbalancer,
			K3sServerArgs:       simpleConfig.Options.K3sOptions.ExtraServerArgs,
			K3sAgentArgs:        simpleConfig.Options.K3sOptions.ExtraAgentArgs,
		},
		ExposeAPI: simpleConfig.ExposeAPI,
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
			node.Ports = append(node.Ports, portWithNodeFilters.Port)
		}
	}

	/******************************
	 * Create Full Cluster Config *
	 ******************************/

	clusterConfig := &ClusterConfig{
		Cluster:           newCluster,
		ClusterCreateOpts: *newCluster.ClusterCreateOpts,
	}

	return clusterConfig, nil
}
