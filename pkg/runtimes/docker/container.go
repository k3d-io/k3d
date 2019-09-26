/*
Copyright © 2019 Thorsten Klein <iwilltry42@gmail.com>

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

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	k3d "github.com/rancher/k3d/pkg/types"
	log "github.com/sirupsen/logrus"
)

// CreateNode creates a new container
func (d Docker) CreateNode(nodeSpec *k3d.Node) error {
	log.Debugln("docker.CreateNode...")
	ctx := context.Background() // TODO: check how kind handles contexts
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("Failed to create docker client. %+v", err)
	}

	containerConfig := container.Config{
		Cmd:   []string{"sh"},
		Image: "nginx",
	}

	resp, err := docker.ContainerCreate(ctx, &containerConfig, &container.HostConfig{}, &network.NetworkingConfig{}, "test")
	if err != nil {
		log.Error("Couldn't create container")
		return err
	}
	log.Infoln("Created", resp.ID)

	return nil
}

// createContainer creates a new docker container
// @return containerID, error
func createContainer(types.Container) (string, error) {
	return "", nil
}

// removeContainer deletes a running container (like docker rm -f)
func removeContainer(ID string) error {

	// (0) create docker client
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("Failed to create docker client. %+v", err)
	}

	// (1) define remove options
	options := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	// (2) remove container
	if err := docker.ContainerRemove(ctx, ID, options); err != nil {
		return fmt.Errorf("Failed to delete container '%s'. %+v", ID, err)
	}

	return nil
}

// DeleteNode deletes a node
func (d Docker) DeleteNode(nodeSpec *k3d.Node) error {
	log.Debugln("docker.DeleteNode...")
	return removeContainer(nodeSpec.Name)
}
