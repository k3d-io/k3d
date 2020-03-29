package container

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"strings"
)

// define struct to expand docker container capabilities
type k3dContainerService struct {
	context 			context.Context
	dockerClient		*client.Client
}

// exported constructor with unexported struct to force initialization by this method
func K3dContainer() (*k3dContainerService, error)  {
	var err error
	k3dContainer := k3dContainerService{}
	k3dContainer.context = context.Background()
	k3dContainer.dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil {
		return nil, fmt.Errorf("Couldn't create docker client\n%+v", err)
	}

	return &k3dContainer, nil
}

// exec command into container. Command is describe by execConfig, target container is containerId
func (k3dContainer *k3dContainerService) ExecIntoContainer(containerId string, execConfig *types.ExecConfig) ([]byte, error) {
	stdoutContent := make([]byte, 0)

	// create exec configuration
	execResponse, err := k3dContainer.dockerClient.ContainerExecCreate(k3dContainer.context, containerId, *execConfig)
	if err != nil {
		return stdoutContent, fmt.Errorf("Failed to create exec command for container [%s]\n%+v", containerId, err)
	}

	// attach to exec process in container
	containerConnection, err := k3dContainer.dockerClient.ContainerExecAttach(k3dContainer.context, execResponse.ID, types.ExecStartCheck{
		Detach: false,
		Tty:    true,
	})
	if err != nil {
		return stdoutContent, fmt.Errorf(" Couldn't attach to container [%s]\n%+v", containerId, err)
	}
	// close connection at function end
	defer containerConnection.Close()

	// start execution
	err = k3dContainer.dockerClient.ContainerExecStart(k3dContainer.context, execResponse.ID, types.ExecStartCheck{
		Tty: true,
	})
	if err != nil {
		return stdoutContent, fmt.Errorf(" Couldn't execute command in container [%s]\n%+v", containerId, err)
	}

	// get output from container
	stdoutContent, err = ioutil.ReadAll(containerConnection.Reader)
	if err != nil {
		return stdoutContent, fmt.Errorf(" Couldn't read output from container [%s]\n%+v", containerId, err)
	}

	// return stdout byte array
	return stdoutContent, nil
}

// get map[string]string of container environment variable
func (k3dContainer *k3dContainerService) GetContainerEnv(containerId string) (map[string]string, error) {
	var envs = make(map[string]string)

	containerInspect, err := k3dContainer.dockerClient.ContainerInspect(k3dContainer.context, containerId)
	if err != nil {
		log.Errorf("Failed to inspect container '%s' to get envs variable", containerId)
		return envs, err
	}

	for _, envVar := range containerInspect.Config.Env {
		envVarSplit := strings.SplitN(envVar, "=", 2)
		envs[envVarSplit[0]] = envVarSplit[1]
	}

	return envs, nil
}