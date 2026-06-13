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

package v1alpha6

import (
	"fmt"

	v1alpha5 "github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
)

// MigrateFromV1Alpha5 migrates a v1alpha5 config to v1alpha6
func MigrateFromV1Alpha5(oldConfig v1alpha5.SimpleConfig) (SimpleConfig, error) {
	newConfig := SimpleConfig{
		TypeMeta:   oldConfig.TypeMeta,
		ObjectMeta: oldConfig.ObjectMeta,
		ExposeAPI: SimpleExposureOpts{
			Host:     oldConfig.ExposeAPI.Host,
			HostIP:   oldConfig.ExposeAPI.HostIP,
			HostPort: oldConfig.ExposeAPI.HostPort,
		},
		Image:        oldConfig.Image,
		Network:      oldConfig.Network,
		Subnet:       oldConfig.Subnet,
		ClusterToken: oldConfig.ClusterToken,
		Options:      migrateOptions(oldConfig.Options),
		Registries: SimpleConfigRegistries{
			Use:    oldConfig.Registries.Use,
			Create: migrateRegistryCreateConfig(oldConfig.Registries.Create),
			Config: oldConfig.Registries.Config,
		},
		HostAliases: oldConfig.HostAliases,
	}

	// Convert servers/agents to nodes
	var nodes []NodeConfig
	
	// Add servers
	for i := 0; i < oldConfig.Servers; i++ {
		node := NodeConfig{
			Role: "server",
		}
		if i == 0 {
			node.Name = fmt.Sprintf("%s-server-0", oldConfig.ObjectMeta.Name)
		}
		nodes = append(nodes, node)
	}
	
	// Add agents
	for i := 0; i < oldConfig.Agents; i++ {
		node := NodeConfig{
			Role: "agent",
		}
		node.Name = fmt.Sprintf("%s-agent-%d", oldConfig.ObjectMeta.Name, i)
		nodes = append(nodes, node)
	}

	// Process volumes with node filters
	for _, vol := range oldConfig.Volumes {
		targetNodes := getTargetNodes(nodes, vol.NodeFilters)
		for i := range targetNodes {
			targetNodes[i].Volumes = append(targetNodes[i].Volumes, vol.Volume)
		}
	}

	// Process ports with node filters
	for _, port := range oldConfig.Ports {
		targetNodes := getTargetNodes(nodes, port.NodeFilters)
		for i := range targetNodes {
			targetNodes[i].Ports = append(targetNodes[i].Ports, port.Port)
		}
	}

	// Process env vars with node filters
	for _, env := range oldConfig.Env {
		targetNodes := getTargetNodes(nodes, env.NodeFilters)
		for i := range targetNodes {
			targetNodes[i].Env = append(targetNodes[i].Env, env.EnvVar)
		}
	}

	// Process labels with node filters
	for _, label := range oldConfig.Options.Runtime.Labels {
		targetNodes := getTargetNodes(nodes, label.NodeFilters)
		for i := range targetNodes {
			targetNodes[i].Labels = append(targetNodes[i].Labels, label.Label)
		}
	}

	// Process k3s args with node filters
	for _, arg := range oldConfig.Options.K3sOptions.ExtraArgs {
		targetNodes := getTargetNodes(nodes, arg.NodeFilters)
		for i := range targetNodes {
			targetNodes[i].ExtraArgs = append(targetNodes[i].ExtraArgs, arg.Arg)
		}
	}

	// Process files with node filters
	for _, file := range oldConfig.Files {
		targetNodes := getTargetNodes(nodes, file.NodeFilters)
		fileConfig := FileConfig{
			Source:      file.Source,
			Destination: file.Destination,
			Description: file.Description,
		}
		for i := range targetNodes {
			targetNodes[i].Files = append(targetNodes[i].Files, fileConfig)
		}
	}

	newConfig.Nodes = nodes
	return newConfig, nil
}

func migrateOptions(oldOptions v1alpha5.SimpleConfigOptions) SimpleConfigOptions {
	return SimpleConfigOptions{
		K3dOptions: SimpleConfigOptionsK3d{
			Wait:                oldOptions.K3dOptions.Wait,
			Timeout:             oldOptions.K3dOptions.Timeout,
			DisableLoadbalancer: oldOptions.K3dOptions.DisableLoadbalancer,
			DisableImageVolume:  oldOptions.K3dOptions.DisableImageVolume,
			NoRollback:          oldOptions.K3dOptions.NoRollback,
			NodeHookActions:     oldOptions.K3dOptions.NodeHookActions,
			Loadbalancer:        migrateLoadbalancerConfig(oldOptions.K3dOptions.Loadbalancer),
		},
		K3sOptions:        migrateK3sOptions(oldOptions.K3sOptions),
		KubeconfigOptions: migrateKubeconfigOptions(oldOptions.KubeconfigOptions),
		Runtime:           migrateRuntimeOptions(oldOptions.Runtime),
	}
}

func migrateK3sOptions(oldK3sOptions v1alpha5.SimpleConfigOptionsK3s) SimpleConfigOptionsK3s {
	var newArgs []K3sArgConfig
	for _, arg := range oldK3sOptions.ExtraArgs {
		newArgs = append(newArgs, K3sArgConfig{Arg: arg.Arg})
	}
	
	var newLabels []string
	for _, label := range oldK3sOptions.NodeLabels {
		newLabels = append(newLabels, label.Label)
	}
	
	return SimpleConfigOptionsK3s{
		ExtraArgs:  newArgs,
		NodeLabels: newLabels,
	}
}

func migrateRuntimeOptions(oldRuntime v1alpha5.SimpleConfigOptionsRuntime) SimpleConfigOptionsRuntime {
	var newLabels []string
	for _, label := range oldRuntime.Labels {
		newLabels = append(newLabels, label.Label)
	}
	
	var newUlimits []Ulimit
	for _, ulimit := range oldRuntime.Ulimits {
		newUlimits = append(newUlimits, Ulimit{
			Name: ulimit.Name,
			Soft: ulimit.Soft,
			Hard: ulimit.Hard,
		})
	}
	
	return SimpleConfigOptionsRuntime{
		GPURequest:    oldRuntime.GPURequest,
		ServersMemory: oldRuntime.ServersMemory,
		AgentsMemory:  oldRuntime.AgentsMemory,
		HostPidMode:   oldRuntime.HostPidMode,
		Labels:        newLabels,
		Ulimits:       newUlimits,
	}
}

func migrateRegistryCreateConfig(old *v1alpha5.SimpleConfigRegistryCreateConfig) *SimpleConfigRegistryCreateConfig {
	if old == nil {
		return nil
	}
	return &SimpleConfigRegistryCreateConfig{
		Name:             old.Name,
		Host:             old.Host,
		HostPort:         old.HostPort,
		Image:            old.Image,
		Proxy:            old.Proxy,
		Volumes:          old.Volumes,
		EnforcePortMatch: old.EnforcePortMatch,
	}
}

func migrateLoadbalancerConfig(old v1alpha5.SimpleConfigOptionsK3dLoadbalancer) SimpleConfigOptionsK3dLoadbalancer {
	return SimpleConfigOptionsK3dLoadbalancer{
		ConfigOverrides: old.ConfigOverrides,
	}
}

func migrateKubeconfigOptions(old v1alpha5.SimpleConfigOptionsKubeconfig) SimpleConfigOptionsKubeconfig {
	return SimpleConfigOptionsKubeconfig{
		UpdateDefaultKubeconfig: old.UpdateDefaultKubeconfig,
		SwitchCurrentContext:    old.SwitchCurrentContext,
	}
}

// getTargetNodes returns the nodes that match the node filters
func getTargetNodes(nodes []NodeConfig, nodeFilters []string) []NodeConfig {
	if len(nodeFilters) == 0 {
		return nodes
	}
	
	var targetNodes []NodeConfig
	for _, filter := range nodeFilters {
		// Parse node filter (e.g., "agents", "servers", "agents:1-2", "server:0")
		if filter == "agents" {
			for _, node := range nodes {
				if node.Role == "agent" {
					targetNodes = append(targetNodes, node)
				}
			}
		} else if filter == "servers" {
			for _, node := range nodes {
				if node.Role == "server" {
					targetNodes = append(targetNodes, node)
				}
			}
		} else {
			// Handle specific node names or indexes
			for _, node := range nodes {
				if node.Name == filter {
					targetNodes = append(targetNodes, node)
				}
			}
		}
	}
	
	return targetNodes
}
