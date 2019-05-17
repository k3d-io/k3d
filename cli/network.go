package run

import (
	"context"
	"fmt"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// createClusterNetwork creates a docker network for a cluster that will be used
// to let the server and worker containers communicate with each other easily.
func createClusterNetwork(clusterName string) (string, error) {
	ctx := context.Background()
	docker, err := client.NewEnvClient()
	if err != nil {
		return "", fmt.Errorf("ERROR: couldn't create docker client\n%+v", err)
	}

	args := filters.NewArgs()
	args.Add("label", "app=k3d")
	args.Add("label", "cluster="+clusterName)
	nl, err := docker.NetworkList(ctx, types.NetworkListOptions{Filters: args})
	if err != nil {
		return "", fmt.Errorf("Failed to list networks\n%+v", err)
	}

	if len(nl) > 1 {
		log.Printf("WARNING: Found %d networks for %s when we only expect 1\n", len(nl), clusterName)
	}

	if len(nl) > 0 {
		return nl[0].ID, nil
	}

	// create the network with a set of labels and the cluster name as network name
	resp, err := docker.NetworkCreate(ctx, clusterName, types.NetworkCreate{
		Labels: map[string]string{
			"app":     "k3d",
			"cluster": clusterName,
		},
	})
	if err != nil {
		return "", fmt.Errorf("ERROR: couldn't create network\n%+v", err)
	}

	return resp.ID, nil
}

// deleteClusterNetwork deletes a docker network based on the name of a cluster it belongs to
func deleteClusterNetwork(clusterName string) error {
	ctx := context.Background()
	docker, err := client.NewEnvClient()
	if err != nil {
		return fmt.Errorf("ERROR: couldn't create docker client\n%+v", err)
	}

	filters := filters.NewArgs()
	filters.Add("label", "app=k3d")
	filters.Add("label", fmt.Sprintf("cluster=%s", clusterName))

	networks, err := docker.NetworkList(ctx, types.NetworkListOptions{
		Filters: filters,
	})
	if err != nil {
		return fmt.Errorf("ERROR: couldn't find network for cluster %s\n%+v", clusterName, err)
	}

	// there should be only one network that matches the name... but who knows?
	for _, network := range networks {
		if err := docker.NetworkRemove(ctx, network.ID); err != nil {
			log.Printf("WARNING: couldn't remove network for cluster %s\n%+v", clusterName, err)
			continue
		}
	}
	return nil
}

// because we do not have support for `host.docker.internal` on linux,
// we'll utilize the Gateway IP of our custom netwerk to reach the host.
func getGatewayIP(clusterName string) (string, error) {
	ctx := context.Background()
	docker, err := client.NewEnvClient()
	if err != nil {
		return "", fmt.Errorf("ERROR: couldn't create docker client\n%+v", err)
	}

	filters := filters.NewArgs()
	filters.Add("label", "app=k3d")
	filters.Add("label", fmt.Sprintf("cluster=%s", clusterName))

	nl, err := docker.NetworkList(ctx, types.NetworkListOptions{
		Filters: filters,
	})
	if err != nil {
		return "", fmt.Errorf("ERROR: couldn't find network for cluster %s\n%+v", clusterName, err)
	}

	if len(nl) > 1 {
		log.Printf("WARNING: Found %d networks for %s when we only expect 1\n", len(nl), clusterName)
	}

	if len(nl) > 0 {
		return nl[0].IPAM.Config[0].Gateway, nil
	}

	return "", fmt.Errorf("ERROR: couldn't find the network entry for cluster %s\n", clusterName)
}