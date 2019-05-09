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
	"github.com/docker/go-connections/nat"
)

type PublishedPorts struct {
	ExposedPorts map[nat.Port]struct{}
	PortBindings   map[nat.Port][]nat.PortBinding
}

// The factory function for PublishedPorts
func createPublishedPorts(specs []string) (*PublishedPorts, error) {
	if  len(specs) == 0 {
		var newExposedPorts = make(map[nat.Port]struct{}, 1)
		var newPortBindings = make(map[nat.Port][]nat.PortBinding, 1)
		return &PublishedPorts{ExposedPorts: newExposedPorts, PortBindings: newPortBindings}, nil
	}

	newExposedPorts, newPortBindings, err := nat.ParsePortSpecs(specs)
	return &PublishedPorts{ExposedPorts: newExposedPorts, PortBindings: newPortBindings}, err
}

// Create a new PublishedPort structure, with all host ports are changed by a fixed 'offset'
func (p PublishedPorts) Offset(offset int) (*PublishedPorts) {
	var newExposedPorts = make(map[nat.Port]struct{}, len(p.ExposedPorts))
	var newPortBindings = make(map[nat.Port][]nat.PortBinding, len(p.PortBindings))

	for k, v := range p.ExposedPorts {
              newExposedPorts[k] = v
	}

	for k, v := range p.PortBindings {
		bindings := make([]nat.PortBinding, len(v))
		for i, b := range v {
			port, _ := nat.ParsePort(b.HostPort)
			bindings[i].HostIP = b.HostIP
			bindings[i].HostPort = fmt.Sprintf("%d",  port + offset)
		}
		newPortBindings[k] = bindings
	}

	return &PublishedPorts{ExposedPorts: newExposedPorts, PortBindings: newPortBindings}
}

// Create a new PublishedPort struct with one more port, based on 'portSpec'
func (p *PublishedPorts) AddPort(portSpec string) (*PublishedPorts, error) {
	portMappings, err := nat.ParsePortSpec(portSpec)
	if err != nil {
		return nil, err
	}

	var newExposedPorts = make(map[nat.Port]struct{}, len(p.ExposedPorts) + 1)
	var newPortBindings = make(map[nat.Port][]nat.PortBinding, len(p.PortBindings) + 1)

	// Populate the new maps
	for k, v := range p.ExposedPorts {
              newExposedPorts[k] = v
	}

	for k, v := range p.PortBindings {
              newPortBindings[k] = v
	}

	// Add new ports
	for _, portMapping := range portMappings {
		port := portMapping.Port
		if _, exists := newExposedPorts[port]; !exists {
			newExposedPorts[port] = struct{}{}
		}

		bslice, exists := newPortBindings[port];
		if !exists {
			bslice = []nat.PortBinding{}
		}
		newPortBindings[port] = append(bslice, portMapping.Binding)
	}

	return &PublishedPorts{ExposedPorts: newExposedPorts, PortBindings: newPortBindings}, nil
}

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

func createServer(verbose bool, image string, port string, args []string, env []string,
                  name string, volumes []string, pPorts *PublishedPorts) (string, error) {
	log.Printf("Creating server using %s...\n", image)

	containerLabels := make(map[string]string)
	containerLabels["app"] = "k3d"
	containerLabels["component"] = "server"
	containerLabels["created"] = time.Now().Format("2006-01-02 15:04:05")
	containerLabels["cluster"] = name

	containerName := fmt.Sprintf("k3d-%s-server", name)

	apiPortSpec := fmt.Sprintf("0.0.0.0:%s:%s/tcp", port, port)
	serverPublishedPorts, err := pPorts.AddPort(apiPortSpec)
	if (err != nil) {
		log.Fatalf("Error: failed to parse API port spec %s \n%+v", apiPortSpec, err)
	}

	hostConfig := &container.HostConfig{
		PortBindings: serverPublishedPorts.PortBindings,
		Privileged: true,
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
		Hostname: containerName,
		Image:    image,
		Cmd:      append([]string{"server"}, args...),
		ExposedPorts: serverPublishedPorts.ExposedPorts,
		Env:    env,
		Labels: containerLabels,
	}
	id, err := startContainer(verbose, config, hostConfig, networkingConfig, containerName)
	if err != nil {
		return "", fmt.Errorf("ERROR: couldn't create container %s\n%+v", containerName, err)
	}

	return id, nil
}

// createWorker creates/starts a k3s agent node that connects to the server
func createWorker(verbose bool, image string, args []string, env []string, name string, volumes []string,
		  postfix int, serverPort string, pPorts *PublishedPorts, publishOffset int) (string, error) {
	containerLabels := make(map[string]string)
	containerLabels["app"] = "k3d"
	containerLabels["component"] = "worker"
	containerLabels["created"] = time.Now().Format("2006-01-02 15:04:05")
	containerLabels["cluster"] = name

	containerName := fmt.Sprintf("k3d-%s-worker-%d", name, postfix)

	env = append(env, fmt.Sprintf("K3S_URL=https://k3d-%s-server:%s", name, serverPort))

	workerPublishedPorts := pPorts.Offset(publishOffset * (postfix + 1))

	hostConfig := &container.HostConfig{
		Tmpfs: map[string]string{
			"/run":     "",
			"/var/run": "",
		},
		PortBindings: workerPublishedPorts.PortBindings,
		Privileged: true,
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
		Hostname: containerName,
		Image:    image,
		Env:      env,
		Labels:   containerLabels,
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
	if err := docker.ContainerRemove(ctx, ID, types.ContainerRemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("FAILURE: couldn't delete container [%s] -> %+v", ID, err)
	}
	return nil
}
