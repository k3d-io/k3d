/*
Copyright © 2020 The k3d Author(s)

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
	"io"
	"io/ioutil"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	log "github.com/sirupsen/logrus"
)

// createContainer creates a new docker container from translated specs
func createContainer(ctx context.Context, dockerNode *NodeInDocker, name string) (string, error) {

	log.Tracef("Creating docker container with translated config\n%+v\n", dockerNode)

	// initialize docker client
	docker, err := GetDockerClient()
	if err != nil {
		log.Errorln("Failed to create docker client")
		return "", err
	}
	defer docker.Close()

	// create container
	var resp container.ContainerCreateCreatedBody
	for {
		resp, err = docker.ContainerCreate(ctx, &dockerNode.ContainerConfig, &dockerNode.HostConfig, &dockerNode.NetworkingConfig, nil, name)
		if err != nil {
			if client.IsErrNotFound(err) {
				if err := pullImage(ctx, docker, dockerNode.ContainerConfig.Image); err != nil {
					log.Errorf("Failed to create container '%s'", name)
					return "", err
				}
				continue
			}
			log.Errorf("Failed to create container '%s'", name)
			return "", err
		}
		log.Debugf("Created container %s (ID: %s)", name, resp.ID)
		break
	}

	return resp.ID, nil
}

func startContainer(ctx context.Context, ID string) error {
	// initialize docker client
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Errorln("Failed to create docker client")
		return err
	}
	defer docker.Close()

	return docker.ContainerStart(ctx, ID, types.ContainerStartOptions{})
}

// removeContainer deletes a running container (like docker rm -f)
func removeContainer(ctx context.Context, ID string) error {

	// (0) create docker client
	docker, err := GetDockerClient()
	if err != nil {
		log.Errorln("Failed to create docker client")
		return err
	}
	defer docker.Close()

	// (1) define remove options
	options := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	// (2) remove container
	if err := docker.ContainerRemove(ctx, ID, options); err != nil {
		log.Errorf("Failed to delete container '%s'", ID)
		return err
	}

	log.Infoln("Deleted", ID)

	return nil
}

// pullImage pulls a container image and outputs progress if --verbose flag is set
func pullImage(ctx context.Context, docker *client.Client, image string) error {

	resp, err := docker.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		log.Errorf("Failed to pull image '%s'", image)
		return err
	}
	defer resp.Close()

	log.Infof("Pulling image '%s'", image)

	// in debug mode (--verbose flag set), output pull progress
	var writer io.Writer = ioutil.Discard
	if log.GetLevel() == log.DebugLevel {
		writer = os.Stdout
	}
	_, err = io.Copy(writer, resp)
	if err != nil {
		log.Warningf("Couldn't get docker output")
		log.Warningln(err)
	}

	return nil

}

func getNodeContainer(ctx context.Context, node *k3d.Node) (*types.Container, error) {
	// (0) create docker client
	docker, err := GetDockerClient()
	if err != nil {
		log.Errorln("Failed to create docker client")
		return nil, err
	}
	defer docker.Close()

	// (1) list containers which have the default k3d labels attached
	filters := filters.NewArgs()
	for k, v := range node.Labels {
		filters.Add("label", fmt.Sprintf("%s=%s", k, v))
	}
	// See https://github.com/moby/moby/issues/29997 for explanation around initial /
	filters.Add("name", fmt.Sprintf("^/?%s$", node.Name)) // regex filtering for exact name match

	containers, err := docker.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters,
		All:     true,
	})
	if err != nil {
		log.Errorln("Failed to list containers")
		return nil, err
	}

	if len(containers) > 1 {
		log.Errorln("Failed to get a single container")
		return nil, err
	}

	if len(containers) == 0 {
		return nil, fmt.Errorf("Didn't find container for node '%s'", node.Name)
	}

	return &containers[0], nil

}
