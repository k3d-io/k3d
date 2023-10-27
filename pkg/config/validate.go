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
	"net/netip"
	"time"

	k3dc "github.com/k3d-io/k3d/v5/pkg/client"
	conf "github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	runtimeutil "github.com/k3d-io/k3d/v5/pkg/runtimes/util"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"

	"fmt"

	dockerunits "github.com/docker/go-units"
)

// ValidateClusterConfig checks a given cluster config for basic errors
func ValidateClusterConfig(ctx context.Context, runtime runtimes.Runtime, config conf.ClusterConfig) error {
	// cluster name must be a valid host name
	if err := k3dc.CheckName(config.Cluster.Name); err != nil {
		return fmt.Errorf("provided cluster name '%s' does not match requirements: %w", config.Cluster.Name, err)
	}

	// network:: edge case: hostnetwork -> only if we have a single node (to avoid port collisions)
	if config.Cluster.Network.Name == "host" && len(config.Cluster.Nodes) > 1 {
		return fmt.Errorf("can only use hostnetwork mode with a single node (port collisions, etc.)")
	}

	// timeout can't be negative
	if config.ClusterCreateOpts.Timeout < 0*time.Second {
		return fmt.Errorf("timeout may not be negative (is '%s')", config.ClusterCreateOpts.Timeout)
	}

	// API-Port cannot be changed when using network=host
	if config.Cluster.Network.Name == "host" && config.Cluster.KubeAPI.Port.Port() != k3d.DefaultAPIPort {
		// in hostNetwork mode, we're not going to map a hostport. Here it should always use 6443.
		// Note that hostNetwork mode is super inflexible and since we don't change the backend port (on the container), it will only be one hostmode cluster allowed.
		return fmt.Errorf("the API Port can not be changed when using 'host' network")
	}

	// memory limits must have proper format
	// if empty we don't care about errors in parsing
	if config.ClusterCreateOpts.ServersMemory != "" {
		if _, err := dockerunits.RAMInBytes(config.ClusterCreateOpts.ServersMemory); err != nil {
			return fmt.Errorf("provided servers memory limit value is invalid: %w", err)
		}
	}

	if config.ClusterCreateOpts.AgentsMemory != "" {
		if _, err := dockerunits.RAMInBytes(config.ClusterCreateOpts.AgentsMemory); err != nil {
			return fmt.Errorf("provided agents memory limit value is invalid: %w", err)
		}
	}

	// hostAliases
	if len(config.ClusterCreateOpts.HostAliases) > 0 {
		// not allowed in hostnetwork mode
		if config.Cluster.Network.Name == "host" {
			return fmt.Errorf("hostAliases not allowed in hostnetwork mode")
		}

		// validate IP and hostname
		for _, ha := range config.ClusterCreateOpts.HostAliases {
			// validate IP
			_, err := netip.ParseAddr(ha.IP)
			if err != nil {
				return fmt.Errorf("invalid IP '%s' in hostAlias '%s': %w", ha.IP, ha, err)
			}

			// validate hostnames
			for _, hostname := range ha.Hostnames {
				if err := k3dc.ValidateHostname(hostname); err != nil {
					return fmt.Errorf("invalid hostname '%s' in hostAlias '%s': %w", hostname, ha, err)
				}
			}
		}
	}

	// validate nodes one by one
	for _, node := range config.Cluster.Nodes {
		// volumes have to be either an existing path on the host or a named runtime volume
		for _, volume := range node.Volumes {
			if err := runtimeutil.ValidateVolumeMount(ctx, runtime, volume, &config.Cluster); err != nil {
				return fmt.Errorf("failed to validate volume mount '%s': %w", volume, err)
			}
		}
	}

	return nil
}
