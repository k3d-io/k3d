package run

/*
 * The functions in this file take care of spinning up the
 * k3s server and worker containers as well as deleting them.
 */

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func createContainer(config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (string, error) {
	ctx := context.Background()

	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("Couldn't create docker client\n%+v", err)
	}

	resp, err := docker.ContainerCreate(ctx, config, hostConfig, networkingConfig, containerName)
	if client.IsErrNotFound(err) {
		log.Printf("Pulling image %s...\n", config.Image)
		reader, err := docker.ImagePull(ctx, config.Image, types.ImagePullOptions{})
		if err != nil {
			return "", fmt.Errorf("Couldn't pull image %s\n%+v", config.Image, err)
		}
		defer reader.Close()
		if ll := log.GetLevel(); ll == log.DebugLevel {
			_, err := io.Copy(os.Stdout, reader)
			if err != nil {
				log.Warningf("Couldn't get docker output\n%+v", err)
			}
		} else {
			_, err := io.Copy(ioutil.Discard, reader)
			if err != nil {
				log.Warningf("Couldn't get docker output\n%+v", err)
			}
		}
		resp, err = docker.ContainerCreate(ctx, config, hostConfig, networkingConfig, containerName)
		if err != nil {
			return "", fmt.Errorf(" Couldn't create container after pull %s\n%+v", containerName, err)
		}
	} else if err != nil {
		return "", fmt.Errorf(" Couldn't create container %s\n%+v", containerName, err)
	}

	return resp.ID, nil
}

func startContainer(ID string) error {
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("Couldn't create docker client\n%+v", err)
	}

	if err := docker.ContainerStart(ctx, ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	return nil
}

func createServer(spec *ClusterSpec) (string, error) {
	log.Printf("Creating server using %s...\n", spec.Image)

	containerLabels := make(map[string]string)
	containerLabels["app"] = "k3d"
	containerLabels["component"] = "server"
	containerLabels["created"] = time.Now().Format("2006-01-02 15:04:05")
	containerLabels["cluster"] = spec.ClusterName

	containerName := GetContainerName("server", spec.ClusterName, -1)

	// labels to be created to the server belong to roles
	// all, server, master or <server-container-name>
	serverLabels, err := MergeLabelSpecs(spec.NodeToLabelSpecMap, "server", containerName)
	if err != nil {
		return "", err
	}
	containerLabels = MergeLabels(containerLabels, serverLabels)

	// ports to be assigned to the server belong to roles
	// all, server, master or <server-container-name>
	serverPorts, err := MergePortSpecs(spec.NodeToPortSpecMap, "server", containerName)
	if err != nil {
		return "", err
	}

	hostIP := "0.0.0.0"
	containerLabels["apihost"] = "localhost"
	if spec.APIPort.Host != "" {
		hostIP = spec.APIPort.HostIP
		containerLabels["apihost"] = spec.APIPort.Host
	}

	apiPortSpec := fmt.Sprintf("%s:%s:%s/tcp", hostIP, spec.APIPort.Port, spec.APIPort.Port)

	serverPorts = append(serverPorts, apiPortSpec)

	serverPublishedPorts, err := CreatePublishedPorts(serverPorts)
	if err != nil {
		log.Fatalf("Error: failed to parse port specs %+v \n%+v", serverPorts, err)
	}

	hostConfig := &container.HostConfig{
		PortBindings: serverPublishedPorts.PortBindings,
		Privileged:   true,
		Init:         &[]bool{true}[0],
	}

	if spec.AutoRestart {
		hostConfig.RestartPolicy.Name = "unless-stopped"
	}

	spec.Volumes.addVolumesToHostConfig(containerName, "server", hostConfig)

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			k3dNetworkName(spec.ClusterName): {
				Aliases: []string{containerName},
			},
		},
	}

	config := &container.Config{
		Hostname:     containerName,
		Image:        spec.Image,
		Cmd:          append([]string{"server"}, spec.ServerArgs...),
		ExposedPorts: serverPublishedPorts.ExposedPorts,
		Env:          spec.Env,
		Labels:       containerLabels,
	}
	id, err := createContainer(config, hostConfig, networkingConfig, containerName)
	if err != nil {
		return "", fmt.Errorf(" Couldn't create container %s\n%+v", containerName, err)
	}

	// copy the registry configuration
	if spec.RegistryEnabled || len(spec.RegistriesFile) > 0 {
		if err := writeRegistriesConfigInContainer(spec, id); err != nil {
			return "", err
		}
	}

	if err := startContainer(id); err != nil {
		return "", fmt.Errorf(" Couldn't start container %s\n%+v", containerName, err)
	}

	return id, nil
}

// createWorker creates/starts a k3s agent node that connects to the server
func createWorker(spec *ClusterSpec, postfix int) (string, error) {
	containerLabels := make(map[string]string)
	containerLabels["app"] = "k3d"
	containerLabels["component"] = "worker"
	containerLabels["created"] = time.Now().Format("2006-01-02 15:04:05")
	containerLabels["cluster"] = spec.ClusterName

	containerName := GetContainerName("worker", spec.ClusterName, postfix)
	env := spec.Env

	needServerURL := true
	for _, envVar := range env {
		if strings.Split(envVar, "=")[0] == "K3S_URL" {
			needServerURL = false
			break
		}
	}
	if needServerURL {
		env = append(spec.Env, fmt.Sprintf("K3S_URL=https://k3d-%s-server:%s", spec.ClusterName, spec.APIPort.Port))
	}

	// labels to be created to the worker belong to roles
	// all, worker, agent or <server-container-name>
	workerLabels, err := MergeLabelSpecs(spec.NodeToLabelSpecMap, "worker", containerName)
	if err != nil {
		return "", err
	}
	containerLabels = MergeLabels(containerLabels, workerLabels)

	// ports to be assigned to the worker belong to roles
	// all, worker, agent or <server-container-name>
	workerPorts, err := MergePortSpecs(spec.NodeToPortSpecMap, "worker", containerName)
	if err != nil {
		return "", err
	}
	workerPublishedPorts, err := CreatePublishedPorts(workerPorts)
	if err != nil {
		return "", err
	}
	if spec.PortAutoOffset > 0 {
		// TODO: add some checks before to print a meaningful log message saying that we cannot map multiple container ports
		// to the same host port without a offset
		workerPublishedPorts = workerPublishedPorts.Offset(postfix + spec.PortAutoOffset)
	}

	hostConfig := &container.HostConfig{
		Tmpfs: map[string]string{
			"/run":     "",
			"/var/run": "",
		},
		PortBindings: workerPublishedPorts.PortBindings,
		Privileged:   true,
		Init:         &[]bool{true}[0],
	}

	if spec.AutoRestart {
		hostConfig.RestartPolicy.Name = "unless-stopped"
	}

	spec.Volumes.addVolumesToHostConfig(containerName, "worker", hostConfig)

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			k3dNetworkName(spec.ClusterName): {
				Aliases: []string{containerName},
			},
		},
	}

	config := &container.Config{
		Hostname:     containerName,
		Image:        spec.Image,
		Env:          env,
		Cmd:          append([]string{"agent"}, spec.AgentArgs...),
		Labels:       containerLabels,
		ExposedPorts: workerPublishedPorts.ExposedPorts,
	}

	id, err := createContainer(config, hostConfig, networkingConfig, containerName)
	if err != nil {
		return "", fmt.Errorf(" Couldn't create container %s\n%+v", containerName, err)
	}

	// copy the registry configuration
	if spec.RegistryEnabled || len(spec.RegistriesFile) > 0 {
		if err := writeRegistriesConfigInContainer(spec, id); err != nil {
			return "", err
		}
	}

	if err := startContainer(id); err != nil {
		return "", fmt.Errorf(" Couldn't start container %s\n%+v", containerName, err)
	}
	return id, nil
}

// removeContainer tries to rm a container, selected by Docker ID, and does a rm -f if it fails (e.g. if container is still running)
func removeContainer(ID string) error {
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf(" Couldn't create docker client\n%+v", err)
	}

	options := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	if err := docker.ContainerRemove(ctx, ID, options); err != nil {
		return fmt.Errorf(" Couldn't delete container [%s] -> %+v", ID, err)
	}
	return nil
}

// getContainerNetworks returns the networks a container is connected to
func getContainerNetworks(ID string) (map[string]*network.EndpointSettings, error) {
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	c, err := docker.ContainerInspect(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf(" Couldn't get details about container %s: %w", ID, err)
	}
	return c.NetworkSettings.Networks, nil
}

// connectContainerToNetwork connects a container to a given network
func connectContainerToNetwork(ID string, networkID string, aliases []string) error {
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf(" Couldn't create docker client\n%+v", err)
	}

	networkingConfig := &network.EndpointSettings{
		Aliases: aliases,
	}

	return docker.NetworkConnect(ctx, networkID, ID, networkingConfig)
}

// disconnectContainerFromNetwork disconnects a container from a given network
func disconnectContainerFromNetwork(ID string, networkID string) error {
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf(" Couldn't create docker client\n%+v", err)
	}

	return docker.NetworkDisconnect(ctx, networkID, ID, false)
}

func waitForContainerLogMessage(containerID string, message string, timeoutSeconds int) error {
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf(" Couldn't create docker client\n%+v", err)
	}

	start := time.Now()
	timeout := time.Duration(timeoutSeconds) * time.Second
	for {
		// not running after timeout exceeded? Rollback and delete everything.
		if timeout != 0 && time.Now().After(start.Add(timeout)) {
			return fmt.Errorf("ERROR: timeout of %d seconds exceeded while waiting for log message '%s'", timeoutSeconds, message)
		}

		// scan container logs for a line that tells us that the required services are up and running
		out, err := docker.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
		if err != nil {
			out.Close()
			return fmt.Errorf("ERROR: couldn't get docker logs from container %s\n%+v", containerID, err)
		}
		buf := new(bytes.Buffer)
		nRead, _ := buf.ReadFrom(out)
		out.Close()
		output := buf.String()
		if nRead > 0 && strings.Contains(string(output), message) {
			break
		}

		time.Sleep(1 * time.Second)
	}
	return nil
}

func copyToContainer(ID string, dstPath string, content []byte) error {
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf(" Couldn't create docker client\n%+v", err)
	}

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	hdr := &tar.Header{Name: dstPath, Mode: 0644, Size: int64(len(content))}
	if err := tw.WriteHeader(hdr); err != nil {
		return errors.Wrap(err, "failed to write a tar header")
	}
	if _, err := tw.Write(content); err != nil {
		return errors.Wrap(err, "failed to write a tar body")
	}
	if err := tw.Close(); err != nil {
		return errors.Wrap(err, "failed to close tar archive")
	}

	r := bytes.NewReader(buf.Bytes())
	if err := docker.CopyToContainer(ctx, ID, "/", r, types.CopyToContainerOptions{AllowOverwriteDirWithFile: true}); err != nil {
		return errors.Wrap(err, "failed to copy source code")
	}
	return nil
}
