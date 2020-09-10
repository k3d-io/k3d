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
	"time"

	"github.com/rancher/k3d/v3/pkg/runtimes"
	"github.com/rancher/k3d/v3/pkg/util"

	k3d "github.com/rancher/k3d/v3/pkg/types"

	k3dc "github.com/rancher/k3d/v3/pkg/cluster"
	log "github.com/sirupsen/logrus"
)

// TransformSimpleToClusterConfig transforms a simple configuration to a full-fledged cluster configuration
func TransformSimpleToClusterConfig(ctx context.Context, runtime runtimes.Runtime, simpleConfig SimpleConfig) (*ClusterConfig, error) {

	// VALIDATE
	if err := k3dc.CheckName(simpleConfig.Name); err != nil {
		log.Errorf("Provided cluster name '%s' does not match requirements", simpleConfig.Name)

		return nil, err
	}

	// TODO: image?

	clusterNetwork := k3d.ClusterNetwork{}
	if simpleConfig.Network != "" {
		clusterNetwork.Name = simpleConfig.Network
		clusterNetwork.External = true
	}

	// network:: edge case: hostnetwork -> only if we have a single node (to avoid port collisions)
	if clusterNetwork.Name == "host" && (simpleConfig.Servers+simpleConfig.Agents) > 1 {
		return nil, fmt.Errorf("Can only use hostnetwork mode with a single node (port collisions, etc.)")
	}

	// TODO: token?

	if simpleConfig.Options.K3dOptions.Timeout < 0*time.Second {
		return nil, fmt.Errorf("Timeout must be > 0s (is '%s')", simpleConfig.Options.K3dOptions.Timeout)
	}

	// -> API
	if simpleConfig.ExposeAPI.Host == "" {
		simpleConfig.ExposeAPI.Host = k3d.DefaultAPIHost
	}
	if simpleConfig.ExposeAPI.HostIP == "" {
		simpleConfig.ExposeAPI.HostIP = k3d.DefaultAPIHost
	}
	if simpleConfig.Network == "host" && simpleConfig.ExposeAPI.Port != k3d.DefaultAPIPort {
		// in hostNetwork mode, we're not going to map a hostport. Here it should always use 6443.
		// Note that hostNetwork mode is super inflexible and since we don't change the backend port (on the container), it will only be one hostmode cluster allowed.
		log.Warnf("Cannot change the API Port when using hostnetwork mode: Falling back to default port '%s'", k3d.DefaultAPIPort)
		simpleConfig.ExposeAPI.Port = k3d.DefaultAPIPort
	}

	// -> VOLUMES
	for _, volumeWithNodeFilters := range simpleConfig.Volumes {
		if err := util.ValidateVolumeMount(runtime, volumeWithNodeFilters.Volume); err != nil {
			return nil, err
		}
	}

	// -> PORTS
	for _, portWithNodeFilters := range simpleConfig.Ports {
		if err := util.ValidatePortMap(portWithNodeFilters.Port); err != nil {
			return nil, err
		}
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

	clusterConfig := &ClusterConfig{
		Cluster:           newCluster,
		ClusterCreateOpts: *newCluster.ClusterCreateOpts,
	}

	// -> NODES
	newCluster.Nodes = []*k3d.Node{}

	if !clusterConfig.ClusterCreateOpts.DisableLoadBalancer {
		newCluster.ServerLoadBalancer = &k3d.Node{
			Role: k3d.LoadBalancerRole,
		}
	}

	return clusterConfig, nil
}
