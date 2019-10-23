package run

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

type networkConfig struct {
	ID      string
	builder func(containerName string) *network.EndpointSettings
}

func k3dNetworkName(clusterName string) string {
	return fmt.Sprintf("k3d-%s", clusterName)
}

// checkIfNetworkExists checks if the specified network exists or not
// if not, an error will be thrown
func checkIfNetworkExists(networkName string) (*networkConfig, error) {
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf(" Couldn't create docker client\n%+v", err)
	}

	args := filters.NewArgs()
	args.Add("name", networkName)
	nl, err := docker.NetworkList(ctx, types.NetworkListOptions{Filters: args})
	if err != nil {
		return nil, fmt.Errorf("Failed to list networks\n%+v", err)
	}

	if len(nl) == 0 {
		return nil, fmt.Errorf("network '%s' doesn't exist, make sure to create it first", networkName)
	}

	endpointBuilder := func(containerName string) *network.EndpointSettings {
		return &network.EndpointSettings{
			Aliases: []string{containerName},
		}
	}

	return &networkConfig{ID: nl[0].ID, builder: endpointBuilder}, nil
}

// createClusterNetwork creates a docker network for a cluster that will be used
// to let the server and worker containers communicate with each other easily.
func createClusterNetwork(clusterName string) (string, error) {
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf(" Couldn't create docker client\n%+v", err)
	}

	args := filters.NewArgs()
	args.Add("label", "app=k3d")
	args.Add("label", "cluster="+clusterName)
	nl, err := docker.NetworkList(ctx, types.NetworkListOptions{Filters: args})
	if err != nil {
		return "", fmt.Errorf("Failed to list networks\n%+v", err)
	}

	if len(nl) > 1 {
		log.Warningf("Found %d networks for %s when we only expect 1\n", len(nl), clusterName)
	}

	if len(nl) > 0 {
		return nl[0].ID, nil
	}

	// create the network with a set of labels and the cluster name as network name
	resp, err := docker.NetworkCreate(ctx, k3dNetworkName(clusterName), types.NetworkCreate{
		Labels: map[string]string{
			"app":     "k3d",
			"cluster": clusterName,
		},
	})
	if err != nil {
		return "", fmt.Errorf(" Couldn't create network\n%+v", err)
	}

	return resp.ID, nil
}

// deleteClusterNetwork deletes a docker network based on the name of a cluster it belongs to
func deleteClusterNetwork(clusterName string) error {
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf(" Couldn't create docker client\n%+v", err)
	}

	filters := filters.NewArgs()
	filters.Add("label", "app=k3d")
	filters.Add("label", fmt.Sprintf("cluster=%s", clusterName))

	networks, err := docker.NetworkList(ctx, types.NetworkListOptions{
		Filters: filters,
	})
	if err != nil {
		return fmt.Errorf(" Couldn't find network for cluster %s\n%+v", clusterName, err)
	}

	// there should be only one network that matches the name... but who knows?
	for _, network := range networks {
		if err := docker.NetworkRemove(ctx, network.ID); err != nil {
			log.Warningf("couldn't remove network for cluster %s\n%+v", clusterName, err)
			continue
		}
	}
	return nil
}
