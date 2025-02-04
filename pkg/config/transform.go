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
	"io"
	"net/netip"
	"os"
	"strings"

	wharfie "github.com/rancher/wharfie/pkg/registries"

	"github.com/docker/go-connections/nat"
	"sigs.k8s.io/yaml"

	dockerunits "github.com/docker/go-units"
	cliutil "github.com/k3d-io/k3d/v5/cmd/util" // TODO: move parseapiport to pkg
	"github.com/k3d-io/k3d/v5/pkg/client"
	conf "github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/pkg/util"
	"github.com/k3d-io/k3d/v5/version"
)

// TransformSimpleToClusterConfig transforms a simple configuration to a full-fledged cluster configuration
func TransformSimpleToClusterConfig(ctx context.Context, runtime runtimes.Runtime, simpleConfig conf.SimpleConfig, configFileName string) (*conf.ClusterConfig, error) {
	// set default cluster name
	if simpleConfig.Name == "" {
		simpleConfig.Name = k3d.DefaultClusterName
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

	for i := 0; i < simpleConfig.Servers; i++ {
		serverNode := k3d.Node{
			Name:        client.GenerateNodeName(newCluster.Name, k3d.ServerRole, i),
			Role:        k3d.ServerRole,
			Image:       simpleConfig.Image,
			ServerOpts:  k3d.ServerOpts{},
			Memory:      simpleConfig.Options.Runtime.ServersMemory,
			HostPidMode: simpleConfig.Options.Runtime.HostPidMode,
		}

		// first server node will be init node if we have more than one server specified but no external datastore
		if i == 0 && simpleConfig.Servers > 1 {
			serverNode.ServerOpts.IsInit = true
			newCluster.InitNode = &serverNode
		}

		newCluster.Nodes = append(newCluster.Nodes, &serverNode)

		if !simpleConfig.Options.K3dOptions.DisableLoadbalancer {
			newCluster.ServerLoadBalancer.Config.Ports[fmt.Sprintf("%s.tcp", k3d.DefaultAPIPort)] = append(newCluster.ServerLoadBalancer.Config.Ports[fmt.Sprintf("%s.tcp", k3d.DefaultAPIPort)], serverNode.Name)
		}
	}

	for i := 0; i < simpleConfig.Agents; i++ {
		agentNode := k3d.Node{
			Name:        client.GenerateNodeName(newCluster.Name, k3d.AgentRole, i),
			Role:        k3d.AgentRole,
			Image:       simpleConfig.Image,
			Memory:      simpleConfig.Options.Runtime.AgentsMemory,
			HostPidMode: simpleConfig.Options.Runtime.HostPidMode,
		}
		newCluster.Nodes = append(newCluster.Nodes, &agentNode)
	}

	/****************************
	 * Extra Node Configuration *
	 ****************************/
	nodeCount := len(newCluster.Nodes)
	nodeList := newCluster.Nodes

	// -> VOLUMES
	for _, volumeWithNodeFilters := range simpleConfig.Volumes {
		nodes, err := util.FilterNodes(nodeList, volumeWithNodeFilters.NodeFilters)
		if err != nil {
			return nil, fmt.Errorf("failed to filter nodes for volume mapping '%s': %w", volumeWithNodeFilters.Volume, err)
		}

		for _, node := range nodes {
			node.Volumes = append(node.Volumes, volumeWithNodeFilters.Volume)
		}
	}

	// -> PORTS
	if err := client.TransformPorts(ctx, runtime, &newCluster, simpleConfig.Ports); err != nil {
		return nil, fmt.Errorf("failed to transform ports: %w", err)
	}

	// -> K3S NODE LABELS
	for _, k3sNodeLabelWithNodeFilters := range simpleConfig.Options.K3sOptions.NodeLabels {
		if len(k3sNodeLabelWithNodeFilters.NodeFilters) == 0 && nodeCount > 1 {
			return nil, fmt.Errorf("k3s node label mapping '%s' lacks a node filter, but there's more than one node", k3sNodeLabelWithNodeFilters.Label)
		}

		nodes, err := util.FilterNodes(nodeList, k3sNodeLabelWithNodeFilters.NodeFilters)
		if err != nil {
			return nil, fmt.Errorf("failed to filter nodes for k3s node label mapping '%s': %w", k3sNodeLabelWithNodeFilters.Label, err)
		}

		for _, node := range nodes {
			if node.K3sNodeLabels == nil {
				node.K3sNodeLabels = make(map[string]string) // ensure that the map is initialized
			}
			k, v := util.SplitLabelKeyValue(k3sNodeLabelWithNodeFilters.Label)
			node.K3sNodeLabels[k] = v
		}
	}

	// -> RUNTIME LABELS
	for _, runtimeLabelWithNodeFilters := range simpleConfig.Options.Runtime.Labels {
		if len(runtimeLabelWithNodeFilters.NodeFilters) == 0 && nodeCount > 1 {
			return nil, fmt.Errorf("RuntimeLabelmapping '%s' lacks a node filter, but there's more than one node", runtimeLabelWithNodeFilters.Label)
		}

		nodes, err := util.FilterNodes(nodeList, runtimeLabelWithNodeFilters.NodeFilters)
		if err != nil {
			return nil, fmt.Errorf("failed to filter nodes for runtime label mapping '%s': %w", runtimeLabelWithNodeFilters.Label, err)
		}

		for _, node := range nodes {
			if node.RuntimeLabels == nil {
				node.RuntimeLabels = make(map[string]string) // ensure that the map is initialized
			}
			k, v := util.SplitLabelKeyValue(runtimeLabelWithNodeFilters.Label)

			cliutil.ValidateRuntimeLabelKey(k)

			node.RuntimeLabels[k] = v
		}
	}

	// -> RUNTIME ULIMITS
	for index, runtimeUlimit := range simpleConfig.Options.Runtime.Ulimits {
		for _, node := range nodeList {
			if node.RuntimeUlimits == nil {
				node.RuntimeUlimits = make([]*dockerunits.Ulimit, len(simpleConfig.Options.Runtime.Ulimits)) // ensure that the map is initialized
			}

			cliutil.ValidateRuntimeUlimitKey(runtimeUlimit.Name)

			node.RuntimeUlimits[index] = &dockerunits.Ulimit{
				Name: runtimeUlimit.Name,
				Soft: runtimeUlimit.Soft,
				Hard: runtimeUlimit.Hard,
			}
		}
	}

	// -> ENV
	for _, envVarWithNodeFilters := range simpleConfig.Env {
		if len(envVarWithNodeFilters.NodeFilters) == 0 && nodeCount > 1 {
			return nil, fmt.Errorf("EnvVarMapping '%s' lacks a node filter, but there's more than one node", envVarWithNodeFilters.EnvVar)
		}

		nodes, err := util.FilterNodes(nodeList, envVarWithNodeFilters.NodeFilters)
		if err != nil {
			return nil, fmt.Errorf("failed to filter nodes for environment variable config '%s': %w", envVarWithNodeFilters.EnvVar, err)
		}

		for _, node := range nodes {
			node.Env = append(node.Env, envVarWithNodeFilters.EnvVar)
		}
	}

	// -> ARGS
	for _, argWithNodeFilters := range simpleConfig.Options.K3sOptions.ExtraArgs {
		if len(argWithNodeFilters.NodeFilters) == 0 && nodeCount > 1 {
			return nil, fmt.Errorf("K3sExtraArg '%s' lacks a node filter, but there's more than one node", argWithNodeFilters.Arg)
		}

		nodes, err := util.FilterNodes(nodeList, argWithNodeFilters.NodeFilters)
		if err != nil {
			return nil, fmt.Errorf("failed to filter nodes for k3s extra args config '%s': %w", argWithNodeFilters.Arg, err)
		}

		for _, node := range nodes {
			node.Args = append(node.Args, argWithNodeFilters.Arg)
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
		GPURequest:          simpleConfig.Options.Runtime.GPURequest,
		ServersMemory:       simpleConfig.Options.Runtime.ServersMemory,
		AgentsMemory:        simpleConfig.Options.Runtime.AgentsMemory,
		HostAliases:         simpleConfig.HostAliases,
		GlobalLabels:        map[string]string{}, // empty init
		GlobalEnv:           []string{},          // empty init
	}

	// ensure, that we have the default object labels
	for k, v := range k3d.DefaultRuntimeLabels {
		clusterCreateOpts.GlobalLabels[k] = v
	}

	/*
	 * Registries
	 */
	if simpleConfig.Registries.Create != nil {
		epSpecHost := "0.0.0.0"
		epSpecPort := "random"

		if simpleConfig.Registries.Create.HostPort != "" {
			epSpecPort = simpleConfig.Registries.Create.HostPort
		}
		if simpleConfig.Registries.Create.Host != "" {
			epSpecHost = simpleConfig.Registries.Create.Host
		}

		regPort, err := cliutil.ParseRegistryPortExposureSpec(fmt.Sprintf("%s:%s", epSpecHost, epSpecPort))
		if err != nil {
			return nil, fmt.Errorf("failed to get port for registry: %w", err)
		}

		regName := fmt.Sprintf("%s-%s-registry", k3d.DefaultObjectNamePrefix, newCluster.Name)
		if simpleConfig.Registries.Create.Name != "" {
			regName = simpleConfig.Registries.Create.Name
		}

		image := fmt.Sprintf("%s:%s", k3d.DefaultRegistryImageRepo, k3d.DefaultRegistryImageTag)
		if simpleConfig.Registries.Create.Image != "" {
			image = simpleConfig.Registries.Create.Image
		}

		clusterCreateOpts.Registries.Create = &k3d.Registry{
			ClusterRef:   newCluster.Name,
			Host:         regName,
			Image:        image,
			ExposureOpts: *regPort,
			Volumes:      simpleConfig.Registries.Create.Volumes,
			Options: k3d.RegistryOptions{
				Proxy: simpleConfig.Registries.Create.Proxy,
			},
		}
	}

	for _, usereg := range simpleConfig.Registries.Use {
		reg, err := util.ParseRegistryRef(usereg)
		if err != nil {
			return nil, fmt.Errorf("failed to parse use-registry string  '%s': %w", usereg, err)
		}
		l.Log().Tracef("Parsed registry reference: %+v", reg)
		clusterCreateOpts.Registries.Use = append(clusterCreateOpts.Registries.Use, reg)
	}

	if simpleConfig.Registries.Config != "" {
		var k3sRegistry *wharfie.Registry

		if strings.Contains(simpleConfig.Registries.Config, "\n") { // CASE 1: embedded registries.yaml (multiline string)
			l.Log().Debugf("Found multiline registries config embedded in SimpleConfig:\n%s", simpleConfig.Registries.Config)
			if err := yaml.Unmarshal([]byte(simpleConfig.Registries.Config), &k3sRegistry); err != nil {
				return nil, fmt.Errorf("failed to read embedded registries config: %w", err)
			}
		} else { // CASE 2: registries.yaml file referenced by path (single line)
			registryConfigFile, err := os.Open(simpleConfig.Registries.Config)
			if err != nil {
				return nil, fmt.Errorf("failed to open registry config file at %s: %w", simpleConfig.Registries.Config, err)
			}
			configBytes, err := io.ReadAll(registryConfigFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read registry config file at %s: %w", registryConfigFile.Name(), err)
			}

			if err := yaml.Unmarshal(configBytes, &k3sRegistry); err != nil {
				return nil, fmt.Errorf("failed to read registry configuration: %w", err)
			}
		}

		l.Log().Tracef("Registry: read config from input:\n%+v", k3sRegistry)
		clusterCreateOpts.Registries.Config = k3sRegistry
	}

	/*
	 * Files
	 */

	for _, fileWithNodeFilters := range simpleConfig.Files {
		nodes, err := util.FilterNodes(nodeList, fileWithNodeFilters.NodeFilters)
		if err != nil {
			return nil, fmt.Errorf("failed to filter nodes for file copying '%s': %w", fileWithNodeFilters, err)
		}

		content, err := util.ReadFileSource(configFileName, fileWithNodeFilters.Source)
		if err != nil {
			return nil, fmt.Errorf("failed to read source content: %w", err)
		}

		destination, err := util.ResolveFileDestination(fileWithNodeFilters.Destination)
		if err != nil {
			return nil, fmt.Errorf("destination path is not correct: %w", err)
		}

		for _, node := range nodes {
			node.Files = append(node.Files, k3d.File{
				Content:     content,
				Destination: destination,
				Description: fileWithNodeFilters.Description,
			})
		}
	}

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
