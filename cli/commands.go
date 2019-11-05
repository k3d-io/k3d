package run

/*
 * This file contains the "backend" functionality for the CLI commands (and flags)
 */

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

const (
	defaultRegistry    = "docker.io"
	defaultServerCount = 1
)

// CheckTools checks if the docker API server is responding
func CheckTools() error {
	log.Print("Checking docker...")
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	ping, err := docker.Ping(ctx)

	if err != nil {
		return fmt.Errorf(" Checking docker failed\n%+v", err)
	}
	log.Printf("SUCCESS: Checking docker succeeded (API: v%s)\n", ping.APIVersion)
	return nil
}

// CreateCluster creates a new single-node cluster container and initializes the cluster directory
func CreateCluster(
	name,
	image string,
	oEnv []string,
	port string,
	serverArgs, agentArgs []string,
	workers int,
	publish, volumes []string,
	autoRestart bool,
	portAutoOffset, wait int,
) error {

	if err := CheckClusterName(name); err != nil {
		return err
	}

	if cluster, err := getClusters(false, name); err != nil {
		return err
	} else if len(cluster) != 0 {
		// A cluster exists with the same name. Return with an error.
		return fmt.Errorf(" Cluster %s already exists", name)
	}

	// On Error delete the cluster.  If there createCluster() encounter any error,
	// call this function to remove all resources allocated for the cluster so far
	// so that they don't linger around.
	deleteCluster := func() {
		if err := DeleteCluster(false, name); err != nil {
			log.Printf("Error: Failed to delete cluster %s", name)
		}
	}

	// define image
	if len(strings.Split(image, "/")) <= 2 {
		// fallback to default registry
		image = fmt.Sprintf("%s/%s", defaultRegistry, image)
	}

	// create cluster network
	networkID, err := createClusterNetwork(name)
	if err != nil {
		return err
	}
	log.Printf("Created cluster network with ID %s", networkID)

	// environment variables
	env := []string{"K3S_KUBECONFIG_OUTPUT=/output/kubeconfig.yaml"}
	env = append(env, oEnv...)
	env = append(env, fmt.Sprintf("K3S_CLUSTER_SECRET=%s", GenerateRandomString(20)))

	// k3s server arguments
	apiPort, err := parseAPIPort(port)
	if err != nil {
		return err
	}

	k3sAgentArgs := []string{}
	k3sServerArgs := []string{"--https-listen-port", apiPort.Port}

	// When the 'host' is not provided by --api-port, try to fill it using Docker Machine's IP address.
	if apiPort.Host == "" {
		apiPort.Host, err = getDockerMachineIp()
		// IP address is the same as the host
		apiPort.HostIP = apiPort.Host
		// In case of error, Log a warning message, and continue on. Since it more likely caused by a miss configured
		// DOCKER_MACHINE_NAME environment variable.
		if err != nil {
			log.Warning("Failed to get docker machine IP address, ignoring the DOCKER_MACHINE_NAME environment variable setting.")
		}
	}

	if apiPort.Host != "" {
		// Add TLS SAN for non default host name
		log.Printf("Add TLS SAN for %s", apiPort.Host)
		k3sServerArgs = append(k3sServerArgs, "--tls-san", apiPort.Host)
	}

	if len(serverArgs) > 0 {
		k3sServerArgs = append(k3sServerArgs, serverArgs...)
	}

	if len(agentArgs) > 0 {
		k3sAgentArgs = append(k3sAgentArgs, agentArgs...)
	}

	// new port map
	portmap, err := mapNodesToPortSpecs(publish, GetAllContainerNames(name, defaultServerCount, workers))
	if err != nil {
		log.Fatal(err)
	}

	// create a docker volume for sharing image tarballs with the cluster
	imageVolume, err := createImageVolume(name)
	log.Println("Created docker volume ", imageVolume.Name)
	if err != nil {
		return err
	}
	volumesSpec, err := NewVolumes(volumes)
	if err != nil {
		return err
	}

	volumesSpec.DefaultVolumes = append(volumesSpec.DefaultVolumes, fmt.Sprintf("%s:/images", imageVolume.Name))

	clusterSpec := &ClusterSpec{
		AgentArgs:         k3sAgentArgs,
		APIPort:           *apiPort,
		AutoRestart:       autoRestart,
		ClusterName:       name,
		Env:               env,
		Image:             image,
		NodeToPortSpecMap: portmap,
		PortAutoOffset:    portAutoOffset,
		ServerArgs:        k3sServerArgs,
		Volumes:           volumesSpec,
	}

	// create the server
	log.Printf("Creating cluster [%s]", name)

	// create the directory where we will put the kubeconfig file by default (when running `k3d get-config`)
	createClusterDir(name)

	dockerID, err := createServer(clusterSpec)
	if err != nil {
		deleteCluster()
		return err
	}

	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	// Wait for k3s to be up and running if wanted.
	// We're simply scanning the container logs for a line that tells us that everything's up and running
	// TODO: also wait for worker nodes
	start := time.Now()
	timeout := time.Duration(wait) * time.Second
	for wait > 0 {
		// not running after timeout exceeded? Rollback and delete everything.
		if timeout != 0 && time.Now().After(start.Add(timeout)) {
			deleteCluster()
			return errors.New("Cluster creation exceeded specified timeout")
		}

		// scan container logs for a line that tells us that the required services are up and running
		out, err := docker.ContainerLogs(ctx, dockerID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
		if err != nil {
			out.Close()
			return fmt.Errorf(" Couldn't get docker logs for %s\n%+v", name, err)
		}
		buf := new(bytes.Buffer)
		nRead, _ := buf.ReadFrom(out)
		out.Close()
		output := buf.String()
		if nRead > 0 && strings.Contains(string(output), "Running kubelet") {
			break
		}

		time.Sleep(1 * time.Second)
	}

	// spin up the worker nodes
	// TODO: do this concurrently in different goroutines
	if workers > 0 {
		log.Printf("Booting %s workers for cluster %s", strconv.Itoa(workers), name)
		for i := 0; i < workers; i++ {
			workerID, err := createWorker(clusterSpec, i)
			if err != nil {
				deleteCluster()
				return err
			}
			log.Printf("Created worker with ID %s\n", workerID)
		}
	}

	log.Printf("SUCCESS: created cluster [%s]", name)
	log.Printf(`You can now use the cluster with:

export KUBECONFIG="$(%s get-kubeconfig --name='%s')"
kubectl cluster-info`, os.Args[0], name)

	return nil
}

// DeleteCluster removes the containers belonging to a cluster and its local directory
func DeleteCluster(all bool, name string) error {
	clusters, err := getClusters(all, name)

	if err != nil {
		return err
	}

	if len(clusters) == 0 {
		return fmt.Errorf("No cluster(s) found")
	}

	// remove clusters one by one instead of appending all names to the docker command
	// this allows for more granular error handling and logging
	for _, cluster := range clusters {
		log.Printf("Removing cluster [%s]", cluster.name)
		if len(cluster.workers) > 0 {
			// TODO: this could be done in goroutines
			log.Printf("...Removing %d workers\n", len(cluster.workers))
			for _, worker := range cluster.workers {
				if err := removeContainer(worker.ID); err != nil {
					log.Println(err)
					continue
				}
			}
		}
		deleteClusterDir(cluster.name)
		log.Println("...Removing server")
		if err := removeContainer(cluster.server.ID); err != nil {
			return fmt.Errorf(" Couldn't remove server for cluster %s\n%+v", cluster.name, err)
		}

		if err := deleteClusterNetwork(cluster.name); err != nil {
			log.Warningf("Couldn't delete cluster network for cluster %s\n%+v", cluster.name, err)
		}

		log.Println("...Removing docker image volume")
		if err := deleteImageVolume(cluster.name); err != nil {
			log.Warningf("Couldn't delete image docker volume for cluster %s\n%+v", cluster.name, err)
		}

		log.Infof("Removed cluster [%s]", cluster.name)
	}

	return nil
}

// StopCluster stops a running cluster container (restartable)
func StopCluster(all bool, name string) error {
	clusters, err := getClusters(all, name)

	if err != nil {
		return err
	}

	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf(" Couldn't create docker client\n%+v", err)
	}

	// remove clusters one by one instead of appending all names to the docker command
	// this allows for more granular error handling and logging
	for _, cluster := range clusters {
		log.Printf("Stopping cluster [%s]", cluster.name)
		if len(cluster.workers) > 0 {
			log.Printf("...Stopping %d workers\n", len(cluster.workers))
			for _, worker := range cluster.workers {
				if err := docker.ContainerStop(ctx, worker.ID, nil); err != nil {
					log.Println(err)
					continue
				}
			}
		}
		log.Println("...Stopping server")
		if err := docker.ContainerStop(ctx, cluster.server.ID, nil); err != nil {
			return fmt.Errorf(" Couldn't stop server for cluster %s\n%+v", cluster.name, err)
		}

		log.Infof("Stopped cluster [%s]", cluster.name)
	}

	return nil
}

// StartCluster starts a stopped cluster container
func StartCluster(all bool, name string) error {
	clusters, err := getClusters(all, name)

	if err != nil {
		return err
	}

	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf(" Couldn't create docker client\n%+v", err)
	}

	// remove clusters one by one instead of appending all names to the docker command
	// this allows for more granular error handling and logging
	for _, cluster := range clusters {
		log.Printf("Starting cluster [%s]", cluster.name)

		log.Println("...Starting server")
		if err := docker.ContainerStart(ctx, cluster.server.ID, types.ContainerStartOptions{}); err != nil {
			return fmt.Errorf(" Couldn't start server for cluster %s\n%+v", cluster.name, err)
		}

		if len(cluster.workers) > 0 {
			log.Printf("...Starting %d workers\n", len(cluster.workers))
			for _, worker := range cluster.workers {
				if err := docker.ContainerStart(ctx, worker.ID, types.ContainerStartOptions{}); err != nil {
					log.Println(err)
					continue
				}
			}
		}

		log.Printf("SUCCESS: Started cluster [%s]", cluster.name)
	}

	return nil
}

// ListClusters prints a list of created clusters
func ListClusters() error {
	if err := printClusters(); err != nil {
		return err
	}
	return nil
}

// GetKubeConfig grabs the kubeconfig from the running cluster and prints the path to stdout
func GetKubeConfig(all bool, name string) error {
	clusters, err := getClusters(all, name)
	if err != nil {
		return err
	}

	if len(clusters) == 0 {
		return fmt.Errorf("No cluster(s) found")
	}

	for _, cluster := range clusters {
		kubeConfigPath, err := getKubeConfig(cluster.name)
		if err != nil {
			if all {
				return err
			}
			log.Println(err)
			continue
		}

		// output kubeconfig file path to stdout
		fmt.Println(kubeConfigPath)
	}

	return nil
}

// Shell starts a new subshell with the KUBECONFIG pointing to the selected cluster
func Shell(name, shell, command string) error {
	return subShell(name, shell, command)
}

// ImportImage saves an image locally and imports it into the k3d containers
func ImportImage(name string, noRemove bool, images []string) error {
	return importImage(name, images, noRemove)
}
