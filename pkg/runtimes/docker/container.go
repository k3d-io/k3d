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
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

// createContainer creates a new docker container from translated specs
func createContainer(dockerNode *NodeInDocker, name string) error {

	log.Debugf("Creating docker container with translated config\n%+v\n", dockerNode) // TODO: remove?

	// initialize docker client
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Errorln("Failed to create docker client")
		return err
	}

	// start container // TODO: check first if image exists locally and pull if it doesn't
	resp, err := docker.ContainerCreate(ctx, &dockerNode.ContainerConfig, &dockerNode.HostConfig, &dockerNode.NetworkingConfig, name)
	if err != nil {
		log.Errorln("Failed to create container")
		return err
	}
	log.Infoln("Created container", resp.ID)

	return nil
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

	log.Infoln("Deleted", ID)

	return nil
}
