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

package config

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	"github.com/docker/go-connections/nat"
	cliutil "github.com/k3d-io/k3d/v5/cmd/util"
	"github.com/k3d-io/k3d/v5/pkg/client"
	confv6 "github.com/k3d-io/k3d/v5/pkg/config/v1alpha6"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/version"
)

// TransformSimpleV1Alpha6ToClusterConfig transforms a v1alpha6 simple configuration to a full-fledged cluster configuration
func TransformSimpleV1Alpha6ToClusterConfig(ctx context.Context, runtime runtimes.Runtime, simpleConfig confv6.SimpleConfig, configFileName string) (*confv6.ClusterConfig, error) {
	// set default cluster name
	if simpleConfig.Name == "" {
		simpleConfig.Name = k3d.DefaultClusterName
	}

	// If no nodes specified, default to one server
	if len(simpleConfig.Nodes) == 0 {
		simpleConfig.Nodes = []confv6.NodeConfig{
			{Role: "server"},
		}
	}

	/* Special cases for Image:
	 * - "latest" / "stable": get latest / stable channel image
	 * - starts with "+": get channel following the "+"
	 */
	if simpleConfig.Image == "latest" || simpleConfig.Image == "stable" || strings.HasPrefix(simpleConfig.Image, "+") {
		searchChannel := strings.TrimPrefix(simpleConfig.Image, "+")
		v, err := version.GetK3sVersion(searchChannel)
		if err != nil {
			return nil, err
		}
		l.Log().Debugf("Using fetched K3s version %s", v)
		simpleConfig.Image = fmt.Sprintf("%s:%s", k3d.DefaultK3sImageRepo, v)
	}

	clusterNetwork := k3d.ClusterNetwork{}
	if simpleConfig.Network != "" {
		clusterNetwork.Name = simpleConfig.Network
		clusterNetwork.External = true
	} else {
		clusterNetwork.Name = fmt.Sprintf("%s-%s", k3d.DefaultObjectNamePrefix, simpleConfig.Name)
		clusterNetwork.External = false
	}

	if simpleConfig.Subnet != "" {
		if simpleConfig.Subnet != "auto" {
			subnet, err := netip.ParsePrefix(simpleConfig.Subnet)
			if err != nil {
				return nil, fmt.Errorf("invalid subnet '%s': %w", simpleConfig.Subnet, err)
			}
			clusterNetwork.IPAM.IPPrefix = subnet
		}
		clusterNetwork.IPAM.Managed = true
	}

	// -> API
	if simpleConfig.ExposeAPI.HostIP == "" {
		simpleConfig.ExposeAPI.HostIP = k3d.DefaultAPIHost
	}
	if simpleConfig.ExposeAPI.Host == "" {
		simpleConfig.ExposeAPI.Host = simpleConfig.ExposeAPI.HostIP
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
		newCluster.ServerLoadBalancer = k3d.NewLoadbalancer()
		lbCreateOpts := &k3d.LoadbalancerCreateOpts{}
		if simpleConfig.Options.K3dOptions.Loadbalancer.ConfigOverrides != nil && len(simpleConfig.Options.K3dOptions.Loadbalancer.ConfigOverrides) > 0 {
			lbCreateOpts.ConfigOverrides = simpleConfig.Options.K3dOptions.Loadbalancer.ConfigOverrides
		}
		var err error
		newCluster.ServerLoadBalancer.Node, err = client.LoadbalancerPrepare(ctx, runtime, &newCluster, lbCreateOpts)
		if err != nil {
			return nil, fmt.Errorf("error preparing the loadbalancer: %w", err)
		}
		newCluster.Nodes = append(newCluster.Nodes, newCluster.ServerLoadBalancer.Node)
	} else {
		l.Log().Debugln("Disabling the load balancer")
	}

	/*************
	 * Add Nodes *
	 *************/

	serverCount := 0
	agentCount := 0

	for _, nodeConfig := range simpleConfig.Nodes {
		// Handle replicas
		replicas := nodeConfig.Replicas
		if replicas == 0 {
			replicas = 1
		}

		for i := 0; i < replicas; i++ {
			var node k3d.Node
			var nodeName string

			if nodeConfig.Name != "" {
				if replicas > 1 {
					nodeName = fmt.Sprintf("%s-%d", nodeConfig.Name, i)
				} else {
					nodeName = nodeConfig.Name
				}
			} else {
				if nodeConfig.Role == "server" {
					nodeName = client.GenerateNodeName(newCluster.Name, k3d.ServerRole, serverCount)
					serverCount++
				} else {
					nodeName = client.GenerateNodeName(newCluster.Name, k3d.AgentRole, agentCount)
					agentCount++
				}
			}

			// Set role
			if nodeConfig.Role == "server" {
				node.Role = k3d.ServerRole
			} else {
				node.Role = k3d.AgentRole
			}

			// Set basic node properties
			node.Name = nodeName
			node.Image = simpleConfig.Image
			if nodeConfig.Image != "" {
				node.Image = nodeConfig.Image
			}

			// Set memory based on role
			if node.Role == k3d.ServerRole {
				node.Memory = simpleConfig.Options.Runtime.ServersMemory
			} else {
				node.Memory = simpleConfig.Options.Runtime.AgentsMemory
			}

			node.HostPidMode = simpleConfig.Options.Runtime.HostPidMode

			// Server-specific settings
			if node.Role == k3d.ServerRole {
				node.ServerOpts = k3d.ServerOpts{}
				
				// first server node will be init node if we have more than one server specified but no external datastore
				if serverCount == 1 && countServers(simpleConfig.Nodes) > 1 {
					node.ServerOpts.IsInit = true
				}
			}

			// Apply node-specific configurations
			if err := applyNodeConfig(&node, nodeConfig, simpleConfig); err != nil {
				return nil, fmt.Errorf("failed to apply config for node %s: %w", node.Name, err)
			}

			newCluster.Nodes = append(newCluster.Nodes, &node)
		}
	}

	// Create cluster config
	clusterConfig := &confv6.ClusterConfig{
		Cluster:       newCluster,
		KubeconfigOpts: simpleConfig.Options.KubeconfigOptions,
	}

	return clusterConfig, nil
}

// applyNodeConfig applies node-specific configuration from the config file
func applyNodeConfig(node *k3d.Node, nodeConfig confv6.NodeConfig, simpleConfig confv6.SimpleConfig) error {
	// Volumes
	if len(nodeConfig.Volumes) > 0 {
		volumeMounts, err := cliutil.ParseVolumeMounts(nodeConfig.Volumes)
		if err != nil {
			return fmt.Errorf("failed to parse volume mounts: %w", err)
		}
		node.Volumes = volumeMounts
	}

	// Ports
	if len(nodeConfig.Ports) > 0 {
		portMappings, err := cliutil.ParsePortMappings(nodeConfig.Ports)
		if err != nil {
			return fmt.Errorf("failed to parse port mappings: %w", err)
		}
		node.Ports = portMappings
	}

	// Environment variables
	if len(nodeConfig.Env) > 0 {
		envVars, err := cliutil.ParseEnvVars(nodeConfig.Env)
		if err != nil {
			return fmt.Errorf("failed to parse environment variables: %w", err)
		}
		node.Env = envVars
	}

	// Labels
	if len(nodeConfig.Labels) > 0 {
		labels, err := cliutil.ParseLabels(nodeConfig.Labels)
		if err != nil {
			return fmt.Errorf("failed to parse labels: %w", err)
		}
		node.Labels = labels
	}

	// Runtime labels (global)
	if len(simpleConfig.Options.Runtime.Labels) > 0 {
		runtimeLabels, err := cliutil.ParseLabels(simpleConfig.Options.Runtime.Labels)
		if err != nil {
			return fmt.Errorf("failed to parse runtime labels: %w", err)
		}
		if node.Labels == nil {
			node.Labels = runtimeLabels
		} else {
			for k, v := range runtimeLabels {
				node.Labels[k] = v
			}
		}
	}

	// K3s extra args
	if len(nodeConfig.ExtraArgs) > 0 {
		node.K3sArgs = nodeConfig.ExtraArgs
	}

	// Global k3s extra args
	if len(simpleConfig.Options.K3sOptions.ExtraArgs) > 0 {
		globalArgs := make([]string, len(simpleConfig.Options.K3sOptions.ExtraArgs))
		for i, arg := range simpleConfig.Options.K3sOptions.ExtraArgs {
			globalArgs[i] = arg.Arg
		}
		if node.K3sArgs == nil {
			node.K3sArgs = globalArgs
		} else {
			node.K3sArgs = append(node.K3sArgs, globalArgs...)
		}
	}

	// Files
	if len(nodeConfig.Files) > 0 {
		for _, file := range nodeConfig.Files {
			node.Files = append(node.Files, k3d.File{
				Source:      file.Source,
				Destination: file.Destination,
				Description: file.Description,
			})
		}
	}

	return nil
}

// countServers counts the total number of server nodes (considering replicas)
func countServers(nodes []confv6.NodeConfig) int {
	count := 0
	for _, node := range nodes {
		if node.Role == "server" {
			replicas := node.Replicas
			if replicas == 0 {
				replicas = 1
			}
			count += replicas
		}
	}
	return count
}
