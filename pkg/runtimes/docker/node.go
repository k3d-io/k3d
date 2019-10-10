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

	"github.com/docker/docker/api/types/filters"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	k3d "github.com/rancher/k3d/pkg/types"
	log "github.com/sirupsen/logrus"
)

// CreateNode creates a new container
func (d Docker) CreateNode(node *k3d.Node) error {
	log.Debugln("docker.CreateNode...")

	// translate node spec to docker container specs
	dockerNode, err := TranslateNodeToContainer(node)
	if err != nil {
		log.Errorln("Failed to translate k3d node specification to docker container specifications")
		return err
	}

	// create node
	if err := createContainer(dockerNode, node.Name); err != nil {
		log.Errorln("Failed to create k3d node")
		return err
	}

	return nil
}

// DeleteNode deletes a node
func (d Docker) DeleteNode(nodeSpec *k3d.Node) error {
	log.Debugln("docker.DeleteNode...")
	return removeContainer(nodeSpec.Name)
}

// GetNodesByLabel returns a list of existing nodes
func (d Docker) GetNodesByLabel(labels map[string]string) ([]*k3d.Node, error) {

	// (0) create docker client
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("Failed to create docker client. %+v", err)
	}

	// (1) list containers which have the default k3d labels attached
	filters := filters.NewArgs()
	for k, v := range k3d.DefaultObjectLabels {
		filters.Add("label", fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range labels {
		filters.Add("label", fmt.Sprintf("%s=%s", k, v))
	}

	containers, err := docker.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters,
		All:     true,
	})
	if err != nil {
		log.Errorln("Failed to list containers")
		return nil, err
	}

	// (2) convert them to node structs
	nodes := []*k3d.Node{}
	for _, container := range containers {
		node := &k3d.Node{
			Name:   container.Names[0],
			Role:   container.Labels["role"], // TODO: catch keyerror
			Labels: container.Labels,
		}
		nodes = append(nodes, node)
	}

	return nodes, nil

}
