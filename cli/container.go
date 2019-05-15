package run

/*
 * The functions in this file take care of spinning up the
 * k3s server and worker containers as well as deleting them.
 */

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

func startContainer(verbose bool, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (string, error) {
	ctx := context.Background()

	docker, err := client.NewEnvClient()
	if err != nil {
		return "", fmt.Errorf("ERROR: couldn't create docker client\n%+v", err)
	}

	resp, err := docker.ContainerCreate(ctx, config, hostConfig, networkingConfig, containerName)
	if client.IsErrImageNotFound(err) {
		log.Printf("Pulling image %s...\n", config.Image)
		reader, err := docker.ImagePull(ctx, config.Image, types.ImagePullOptions{})
		if err != nil {
			return "", fmt.Errorf("ERROR: couldn't pull image %s\n%+v", config.Image, err)
		}
		defer reader.Close()
		if verbose {
			_, err := io.Copy(os.Stdout, reader)
			if err != nil {
				log.Printf("WARNING: couldn't get docker output\n%+v", err)
			}
		} else {
			_, err := io.Copy(ioutil.Discard, reader)
			if err != nil {
				log.Printf("WARNING: couldn't get docker output\n%+v", err)
			}
		}
		resp, err = docker.ContainerCreate(ctx, config, hostConfig, networkingConfig, containerName)
		if err != nil {
			return "", fmt.Errorf("ERROR: couldn't create container after pull %s\n%+v", containerName, err)
		}
	} else if err != nil {
		return "", fmt.Errorf("ERROR: couldn't create container %s\n%+v", containerName, err)
	}

	if err := docker.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	return resp.ID, nil
}

func createServer(verbose bool, image string, apiPort string, args []string, env []string,
	name string, volumes []string, nodeToPortSpecMap map[string][]string) (string, error) {
	log.Printf("Creating server using %s...\n", image)

	containerLabels := make(map[string]string)
	containerLabels["app"] = "k3d"
	containerLabels["component"] = "server"
	containerLabels["created"] = time.Now().Format("2006-01-02 15:04:05")
	containerLabels["cluster"] = name

	containerName := GetContainerName("server", name, -1)

	// ports to be assigned to the server belong to roles
	// all, server or <server-container-name>
	serverPorts, err := MergePortSpecs(nodeToPortSpecMap, "server", containerName)
	if err != nil {
		return "", err
	}

	apiPortSpec := fmt.Sprintf("0.0.0.0:%s:%s/tcp", apiPort, apiPort)

	serverPorts = append(serverPorts, apiPortSpec)

	serverPublishedPorts, err := CreatePublishedPorts(serverPorts)
	if err != nil {
		log.Fatalf("Error: failed to parse port specs %+v \n%+v", serverPorts, err)
	}

	hostConfig := &container.HostConfig{
		PortBindings: serverPublishedPorts.PortBindings,
		Privileged:   true,
	}

	if len(volumes) > 0 && volumes[0] != "" {
		hostConfig.Binds = volumes
	}

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			name: {
				Aliases: []string{containerName},
			},
		},
	}

	config := &container.Config{
		Hostname:     containerName,
		Image:        image,
		Cmd:          append([]string{"server"}, args...),
		ExposedPorts: serverPublishedPorts.ExposedPorts,
		Env:          env,
		Labels:       containerLabels,
	}
	id, err := startContainer(verbose, config, hostConfig, networkingConfig, containerName)
	if err != nil {
		return "", fmt.Errorf("ERROR: couldn't create container %s\n%+v", containerName, err)
	}

	return id, nil
}

// createWorker creates/starts a k3s agent node that connects to the server
func createWorker(verbose bool, image string, args []string, env []string, name string, volumes []string,
	postfix int, serverPort string, nodeToPortSpecMap map[string][]string, portAutoOffset int) (string, error) {
	containerLabels := make(map[string]string)
	containerLabels["app"] = "k3d"
	containerLabels["component"] = "worker"
	containerLabels["created"] = time.Now().Format("2006-01-02 15:04:05")
	containerLabels["cluster"] = name

	containerName := GetContainerName("worker", name, postfix)

	env = append(env, fmt.Sprintf("K3S_URL=https://k3d-%s-server:%s", name, serverPort))

	// ports to be assigned to the server belong to roles
	// all, server or <server-container-name>
	workerPorts, err := MergePortSpecs(nodeToPortSpecMap, "worker", containerName)
	if err != nil {
		return "", err
	}
	workerPublishedPorts, err := CreatePublishedPorts(workerPorts)
	if err != nil {
		return "", err
	}
	if portAutoOffset > 0 {
		// TODO: add some checks before to print a meaningful log message saying that we cannot map multiple container ports
		// to the same host port without a offset
		workerPublishedPorts = workerPublishedPorts.Offset(postfix + portAutoOffset)
	}

	hostConfig := &container.HostConfig{
		Tmpfs: map[string]string{
			"/run":     "",
			"/var/run": "",
		},
		PortBindings: workerPublishedPorts.PortBindings,
		Privileged:   true,
	}

	if len(volumes) > 0 && volumes[0] != "" {
		hostConfig.Binds = volumes
	}

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			name: {
				Aliases: []string{containerName},
			},
		},
	}

	config := &container.Config{
		Hostname:     containerName,
		Image:        image,
		Env:          env,
		Labels:       containerLabels,
		ExposedPorts: workerPublishedPorts.ExposedPorts,
	}

	id, err := startContainer(verbose, config, hostConfig, networkingConfig, containerName)
	if err != nil {
		return "", fmt.Errorf("ERROR: couldn't start container %s\n%+v", containerName, err)
	}

	return id, nil
}

// removeContainer tries to rm a container, selected by Docker ID, and does a rm -f if it fails (e.g. if container is still running)
func removeContainer(ID string) error {
	ctx := context.Background()
	docker, err := client.NewEnvClient()
	if err != nil {
		return fmt.Errorf("ERROR: couldn't create docker client\n%+v", err)
	}

	options := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:true,
	}

	if err := docker.ContainerRemove(ctx, ID, options); err != nil {
		return fmt.Errorf("FAILURE: couldn't delete container [%s] -> %+v", ID, err)
	}
	return nil
}
