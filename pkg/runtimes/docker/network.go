/*
Copyright © 2019 Thorsten Klein <iwilltry42@gmail.com>

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
package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	k3d "github.com/rancher/k3d/pkg/types"
	log "github.com/sirupsen/logrus"
)

// CreateNetworkIfNotExist creates a new docker network
// @return networkID, error
func CreateNetworkIfNotExist(clusterName string) (string, error) {

	networkName := k3d.GetDefaultObjectName(clusterName)

	// (0) create new docker client
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("Failed to create docker client. %+v", err)
	}

	// (1) configure list filters
	args := GetDefaultObjectLabelsFilter(clusterName)
	args.Add("name", networkName)
	// TODO: filter for label: cluster=<clusterName>?

	// (2) get filtered list of networks
	networkList, err := docker.NetworkList(ctx, types.NetworkListOptions{
		Filters: args,
	})
	if err != nil {
		return "", fmt.Errorf("Failed to list docker networks. %+v", err)
	}

	// (2.1) If possible, return an existing network
	if len(networkList) > 1 {
		log.Warnf("Found %d networks instead of only one. Choosing the first one: '%s'.", len(networkList), networkList[0].ID)
	}

	if len(networkList) > 0 {
		return networkList[0].ID, nil
	}

	// (3) Create a new network
	// (3.1) Define network labels
	labels := k3d.DefaultObjectLabels
	labels["cluster"] = clusterName

	// (3.2) Create network
	network, err := docker.NetworkCreate(ctx, networkName, types.NetworkCreate{
		Labels: k3d.DefaultObjectLabels,
	})
	if err != nil {
		return "", fmt.Errorf("Failed to create network. %+v", err)
	}

	return network.ID, nil
}
