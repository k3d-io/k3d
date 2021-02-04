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
package docker

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	runtimeErr "github.com/rancher/k3d/v4/pkg/runtimes/errors"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	log "github.com/sirupsen/logrus"
)

// CreateNetworkIfNotPresent creates a new docker network
// @return: network name, exists, error
func (d Docker) CreateNetworkIfNotPresent(ctx context.Context, name string) (string, bool, error) {

	// (0) create new docker client
	docker, err := GetDockerClient()
	if err != nil {
		log.Errorln("Failed to create docker client")
		return "", false, err
	}
	defer docker.Close()

	// (1) configure list filters
	filter := filters.NewArgs()
	filter.Add("name", fmt.Sprintf("^/?%s$", name)) // regex filtering for exact name match

	// (2) get filtered list of networks
	networkList, err := docker.NetworkList(ctx, types.NetworkListOptions{
		Filters: filter,
	})
	if err != nil {
		log.Errorln("Failed to list docker networks")
		return "", false, err
	}

	// (2.1) If possible, return an existing network
	if len(networkList) > 1 {
		log.Warnf("Found %d networks instead of only one. Choosing the first one: '%s'.", len(networkList), networkList[0].ID)
	}

	if len(networkList) > 0 {
		log.Infof("Network with name '%s' already exists with ID '%s'", name, networkList[0].ID)
		return networkList[0].ID, true, nil
	}

	// (3) Create a new network
	network, err := docker.NetworkCreate(ctx, name, types.NetworkCreate{
		Labels: k3d.DefaultObjectLabels,
	})
	if err != nil {
		log.Errorln("Failed to create network")
		return "", false, err
	}

	log.Infof("Created network '%s'", name)
	return network.ID, false, nil
}

// DeleteNetwork deletes a network
func (d Docker) DeleteNetwork(ctx context.Context, ID string) error {
	// (0) create new docker client
	docker, err := GetDockerClient()
	if err != nil {
		log.Errorln("Failed to create docker client")
		return err
	}
	defer docker.Close()

	// (3) delete network
	if err := docker.NetworkRemove(ctx, ID); err != nil {
		if strings.HasSuffix(err.Error(), "active endpoints") {
			return runtimeErr.ErrRuntimeNetworkNotEmpty
		}
		return err
	}
	return nil
}

// GetNetwork gets information about a network by its ID
func GetNetwork(ctx context.Context, ID string) (types.NetworkResource, error) {
	docker, err := GetDockerClient()
	if err != nil {
		log.Errorln("Failed to create docker client")
		return types.NetworkResource{}, err
	}
	defer docker.Close()
	return docker.NetworkInspect(ctx, ID, types.NetworkInspectOptions{})
}

// GetGatewayIP returns the IP of the network gateway
func GetGatewayIP(ctx context.Context, network string) (net.IP, error) {
	bridgeNetwork, err := GetNetwork(ctx, network)
	if err != nil {
		log.Errorf("Failed to get bridge network with name '%s'", network)
		return nil, err
	}

	gatewayIP := net.ParseIP(bridgeNetwork.IPAM.Config[0].Gateway)

	return gatewayIP, nil
}

// ConnectNodeToNetwork connects a node to a network
func (d Docker) ConnectNodeToNetwork(ctx context.Context, node *k3d.Node, networkName string) error {
	// get container
	container, err := getNodeContainer(ctx, node)
	if err != nil {
		return err
	}

	// get docker client
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Errorln("Failed to create docker client")
		return err
	}
	defer docker.Close()

	// get network
	networkResource, err := GetNetwork(ctx, networkName)
	if err != nil {
		log.Errorf("Failed to get network '%s'", networkName)
		return err
	}

	// connect container to network
	return docker.NetworkConnect(ctx, networkResource.ID, container.ID, &network.EndpointSettings{})
}

// DisconnectNodeFromNetwork disconnects a node from a network (u don't say :O)
func (d Docker) DisconnectNodeFromNetwork(ctx context.Context, node *k3d.Node, networkName string) error {
	log.Debugf("Disconnecting node %s from network %s...", node.Name, networkName)
	// get container
	container, err := getNodeContainer(ctx, node)
	if err != nil {
		return err
	}

	// get docker client
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Errorln("Failed to create docker client")
		return err
	}
	defer docker.Close()

	// get network
	networkResource, err := GetNetwork(ctx, networkName)
	if err != nil {
		log.Errorf("Failed to get network '%s'", networkName)
		return err
	}

	return docker.NetworkDisconnect(ctx, networkResource.ID, container.ID, true)
}
