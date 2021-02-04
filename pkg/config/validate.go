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
	"time"

	k3dc "github.com/rancher/k3d/v4/pkg/client"
	conf "github.com/rancher/k3d/v4/pkg/config/v1alpha2"
	"github.com/rancher/k3d/v4/pkg/runtimes"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	"github.com/rancher/k3d/v4/pkg/util"

	"fmt"

	log "github.com/sirupsen/logrus"
)

// ValidateClusterConfig checks a given cluster config for basic errors
func ValidateClusterConfig(ctx context.Context, runtime runtimes.Runtime, config conf.ClusterConfig) error {
	// cluster name must be a valid host name
	if err := k3dc.CheckName(config.Cluster.Name); err != nil {
		log.Errorf("Provided cluster name '%s' does not match requirements", config.Cluster.Name)

		return err
	}

	// network:: edge case: hostnetwork -> only if we have a single node (to avoid port collisions)
	if config.Cluster.Network.Name == "host" && len(config.Cluster.Nodes) > 1 {
		return fmt.Errorf("Can only use hostnetwork mode with a single node (port collisions, etc.)")
	}

	// timeout can't be negative
	if config.ClusterCreateOpts.Timeout < 0*time.Second {
		return fmt.Errorf("Timeout may not be negative (is '%s')", config.ClusterCreateOpts.Timeout)
	}

	// API-Port cannot be changed when using network=host
	if config.Cluster.Network.Name == "host" && config.Cluster.KubeAPI.Port.Port() != k3d.DefaultAPIPort {
		// in hostNetwork mode, we're not going to map a hostport. Here it should always use 6443.
		// Note that hostNetwork mode is super inflexible and since we don't change the backend port (on the container), it will only be one hostmode cluster allowed.
		return fmt.Errorf("The API Port can not be changed when using 'host' network")
	}

	// validate nodes one by one
	for _, node := range config.Cluster.Nodes {

		// node names have to be valid hostnames // TODO: validate hostnames once we generate them before this step
		/*if err := k3dc.CheckName(node.Name); err != nil {
			return err
		}*/

		// volumes have to be either an existing path on the host or a named runtime volume
		for _, volume := range node.Volumes {

			if err := util.ValidateVolumeMount(runtime, volume); err != nil {
				return err
			}
		}
	}

	return nil
}
