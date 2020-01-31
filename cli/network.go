package run

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

func k3dNetworkName(clusterName string) string {
	return fmt.Sprintf("k3d-%s", clusterName)
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

func getClusterNetwork(clusterName string) (string, error) {
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf(" Couldn't create docker client\n%+v", err)
	}

	filters := filters.NewArgs()
	filters.Add("label", "app=k3d")
	filters.Add("label", fmt.Sprintf("cluster=%s", clusterName))

	networks, err := docker.NetworkList(ctx, types.NetworkListOptions{
		Filters: filters,
	})
	if err != nil {
		return "", fmt.Errorf(" Couldn't find network for cluster %s\n%+v", clusterName, err)
	}
	if len(networks) == 0 {
		return "", nil
	}
	// there should be only one network that matches the name... but who knows?
	return networks[0].ID, nil
}

// deleteClusterNetwork deletes a docker network based on the name of a cluster it belongs to
func deleteClusterNetwork(clusterName string) error {
	nid, err := getClusterNetwork(clusterName)
	if err != nil {
		return fmt.Errorf(" Couldn't find network for cluster %s\n%+v", clusterName, err)
	}
	if nid == "" {
		log.Warningf("couldn't remove network for cluster %s: network does not exist", clusterName)
		return nil
	}

	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf(" Couldn't create docker client\n%+v", err)
	}
	if err := docker.NetworkRemove(ctx, nid); err != nil {
		log.Warningf("couldn't remove network for cluster %s\n%+v", clusterName, err)
	}
	return nil
}

// getContainersInNetwork gets a list of containers connected to a network
func getContainersInNetwork(nid string) ([]string, error) {
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("Couldn't create docker client\n%+v", err)
	}

	options := types.NetworkInspectOptions{}
	network, err := docker.NetworkInspect(ctx, nid, options)
	if err != nil {
		return nil, err
	}
	cids := []string{}
	for cid := range network.Containers {
		cids = append(cids, cid)
	}
	return cids, nil
}
