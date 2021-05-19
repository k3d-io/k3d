/*
Copyright © 2020-2021 The k3d Author(s)

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
	"inet.af/netaddr"

	runtimeErr "github.com/rancher/k3d/v4/pkg/runtimes/errors"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	"github.com/rancher/k3d/v4/pkg/util"
	log "github.com/sirupsen/logrus"
)

// GetNetwork returns a given network
func (d Docker) GetNetwork(ctx context.Context, searchNet *k3d.ClusterNetwork) (*k3d.ClusterNetwork, error) {
	// (0) create new docker client
	docker, err := GetDockerClient()
	if err != nil {
		log.Errorln("Failed to create docker client")
		return nil, err
	}
	defer docker.Close()

	if searchNet.ID == "" && searchNet.Name == "" {
		return nil, fmt.Errorf("need one of name, id to get network")
	}
	// configure list filters
	filter := filters.NewArgs()
	if searchNet.ID != "" {
		filter.Add("id", fmt.Sprintf("^/?%s$", searchNet.ID)) // regex filtering for exact ID match
	}
	if searchNet.Name != "" {
		filter.Add("name", fmt.Sprintf("^/?%s$", searchNet.Name)) // regex filtering for exact name match
	}

	// get filtered list of networks
	networkList, err := docker.NetworkList(ctx, types.NetworkListOptions{
		Filters: filter,
	})
	if err != nil {
		log.Errorln("Failed to list docker networks")
		return nil, err
	}

	if len(networkList) == 0 {
		return nil, runtimeErr.ErrRuntimeNetworkNotExists
	}

	network := &k3d.ClusterNetwork{
		Name: networkList[0].Name,
		ID:   networkList[0].ID,
	}

	// for networks that have an IPAM config, we inspect that as well (e.g. "host" network doesn't have it)
	if len(networkList[0].IPAM.Config) > 0 {
		prefix, err := netaddr.ParseIPPrefix(networkList[0].IPAM.Config[0].Subnet)
		if err != nil {
			return nil, err
		}

		gateway, err := netaddr.ParseIP(networkList[0].IPAM.Config[0].Gateway)
		if err != nil {
			return nil, err
		}

		network.IPAM = k3d.IPAM{
			IPPrefix: prefix,
			IPsUsed: []netaddr.IP{
				gateway,
			},
		}

		for _, container := range networkList[0].Containers {
			if container.IPv4Address != "" {
				ip, err := netaddr.ParseIP(container.IPv4Address)
				if err != nil {
					return nil, err
				}
				network.IPAM.IPsUsed = append(network.IPAM.IPsUsed, ip)
			}
		}

		// append the used IPs that we already know from the search network
		// this is needed because the network inspect does not return the container list until the containers are actually started
		// and we already need this when we create the containers
		for _, used := range searchNet.IPAM.IPsUsed {
			network.IPAM.IPsUsed = append(network.IPAM.IPsUsed, used)
		}
	} else {
		log.Debugf("Network %s does not have an IPAM config", network.Name)
	}

	// Only one Network allowed, but some functions don't care about this, so they can ignore the error and just use the first one returned
	if len(networkList) > 1 {
		return network, runtimeErr.ErrRuntimeNetworkMultiSameName
	}

	return network, nil

}

// CreateNetworkIfNotPresent creates a new docker network
// @return: network, exists, error
func (d Docker) CreateNetworkIfNotPresent(ctx context.Context, inNet *k3d.ClusterNetwork) (*k3d.ClusterNetwork, bool, error) {

	// (0) create new docker client
	docker, err := GetDockerClient()
	if err != nil {
		log.Errorln("Failed to create docker client")
		return nil, false, err
	}
	defer docker.Close()

	existingNet, err := d.GetNetwork(ctx, inNet)
	if err != nil {
		if err != runtimeErr.ErrRuntimeNetworkNotExists {
			if existingNet == nil {
				log.Errorln("Failed to check for duplicate networks")
				return nil, false, err
			} else if err == runtimeErr.ErrRuntimeNetworkMultiSameName {
				log.Warnf("%+v, returning the first one: %s (%s)", err, existingNet.Name, existingNet.ID)
				return existingNet, true, nil
			} else {
				return nil, false, fmt.Errorf("unhandled error while checking for existing networks: %+v", err)
			}
		}
	}
	if existingNet != nil {
		return existingNet, true, nil
	}

	// (3) Create a new network
	netCreateOpts := types.NetworkCreate{
		CheckDuplicate: true,
		Labels:         k3d.DefaultRuntimeLabels,
	}

	// we want a managed (user-defined) network, but user didn't specify a subnet, so we try to auto-generate one
	if inNet.IPAM.Managed && inNet.IPAM.IPPrefix.IsZero() {
		log.Traceln("No subnet prefix given, but network should be managed: Trying to get a free subnet prefix...")
		freeSubnetPrefix, err := d.getFreeSubnetPrefix(ctx)
		if err != nil {
			return nil, false, err
		}
		inNet.IPAM.IPPrefix = freeSubnetPrefix
	}

	// use user-defined subnet, if given
	if !inNet.IPAM.IPPrefix.IsZero() {
		netCreateOpts.IPAM = &network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet:  inNet.IPAM.IPPrefix.String(),
					Gateway: inNet.IPAM.IPPrefix.Range().From.Next().String(), // second IP in subnet will be the Gateway (Next, so we don't hit x.x.x.0)
				},
			},
		}
	}

	newNet, err := docker.NetworkCreate(ctx, inNet.Name, netCreateOpts)
	if err != nil {
		log.Errorln("Failed to create new network")
		return nil, false, err
	}

	networkDetails, err := docker.NetworkInspect(ctx, newNet.ID, types.NetworkInspectOptions{})
	if err != nil {
		log.Errorln("Failed to inspect newly created network")
		return nil, false, err
	}

	log.Infof("Created network '%s' (%s)", inNet.Name, networkDetails.ID)
	prefix, err := netaddr.ParseIPPrefix(networkDetails.IPAM.Config[0].Subnet)
	if err != nil {
		return nil, false, err
	}

	newClusterNet := &k3d.ClusterNetwork{Name: inNet.Name, ID: networkDetails.ID, IPAM: k3d.IPAM{IPPrefix: prefix}}

	if !inNet.IPAM.IPPrefix.IsZero() {
		newClusterNet.IPAM.Managed = true
	}

	return newClusterNet, false, nil
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

	if len(bridgeNetwork.IPAM.Config) > 0 {
		gatewayIP := net.ParseIP(bridgeNetwork.IPAM.Config[0].Gateway)
		return gatewayIP, nil
	} else {
		return nil, fmt.Errorf("Failed to get IPAM Config for network %s", bridgeNetwork.Name)
	}

}

// ConnectNodeToNetwork connects a node to a network
func (d Docker) ConnectNodeToNetwork(ctx context.Context, node *k3d.Node, networkName string) error {
	// check that node was not attached to network before
	if isAttachedToNetwork(node, networkName) {
		log.Infof("Container '%s' is already connected to '%s'", node.Name, networkName)
		return nil
	}

	// get container
	container, err := getNodeContainer(ctx, node)
	if err != nil {
		return err
	}

	// get docker client
	docker, err := GetDockerClient()
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
	docker, err := GetDockerClient()
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

func (d Docker) getFreeSubnetPrefix(ctx context.Context) (netaddr.IPPrefix, error) {
	// (0) create new docker client
	docker, err := GetDockerClient()
	if err != nil {
		return netaddr.IPPrefix{}, fmt.Errorf("failed to create docker client %w", err)
	}
	defer docker.Close()

	// 1. Create a fake network to get auto-generated subnet prefix
	fakenetName := fmt.Sprintf("%s-fakenet-%s", k3d.DefaultObjectNamePrefix, util.GenerateRandomString(10))
	fakenetResp, err := docker.NetworkCreate(ctx, fakenetName, types.NetworkCreate{})
	if err != nil {
		return netaddr.IPPrefix{}, fmt.Errorf("failed to create fake network: %w", err)
	}

	fakenet, err := d.GetNetwork(ctx, &k3d.ClusterNetwork{ID: fakenetResp.ID})
	if err != nil {
		return netaddr.IPPrefix{}, fmt.Errorf("failed to inspect fake network %s: %w", fakenetResp.ID, err)
	}

	log.Tracef("Created fake network %s (%s) with subnet prefix %s. Deleting it again to re-use that prefix...", fakenet.Name, fakenet.ID, fakenet.IPAM.IPPrefix.String())

	if err := d.DeleteNetwork(ctx, fakenet.ID); err != nil {
		return netaddr.IPPrefix{}, fmt.Errorf("failed to delete fake network %s (%s): %w", fakenet.Name, fakenet.ID, err)
	}

	return fakenet.IPAM.IPPrefix, nil

}
