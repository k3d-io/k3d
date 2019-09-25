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

package containerd

import (
	"context"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/containers"
	k3d "github.com/rancher/k3d/pkg/types"
	log "github.com/sirupsen/logrus"
)

// CreateNode creates a new k3d node
func (d Containerd) CreateNode(nodeSpec *k3d.Node) error {
	log.Debugln("containerd.CreateNode...")
	ctx := context.Background()
	clientOpts := []containerd.ClientOpt{
		containerd.WithDefaultNamespace("k3d"),
	}
	client, err := containerd.New("/run/containerd/containerd.sock", clientOpts...) // TODO: this is the default address on UNIX, different on Windows
	if err != nil {
		log.Errorln("Failed to create containerd client")
		return err
	}
	newContainerOpts := []containerd.NewContainerOpts{
		func(ctx context.Context, _ *containerd.Client, c *containers.Container) error {
			c.Image = "docker.io/nginx:latest"
			c.Labels = map[string]string{
				"runtime": "containerd",
			}
			return nil
		},
	}
	resp, err := client.NewContainer(ctx, "test-containerd", newContainerOpts...)
	if err != nil {
		log.Errorln("Couldn't create container")
		return err
	}
	log.Debugln("Created container with ID", resp.ID)
	return nil
}

// DeleteNode deletes an existing k3d node
func (d Containerd) DeleteNode(nodeSpec *k3d.Node) error {
	log.Debugln("containerd.DeleteNode...")
	ctx := context.Background()
	clientOpts := []containerd.ClientOpt{
		containerd.WithDefaultNamespace("k3d"),
	}
	client, err := containerd.New("/run/containerd/containerd.sock", clientOpts...) // TODO: this is the default address on UNIX, different on Windows
	if err != nil {
		log.Errorln("Failed to create containerd client")
		return err
	}

	container, err := client.LoadContainer(ctx, nodeSpec.Name)
	if err != nil {
		log.Errorln("Couldn't load container", nodeSpec.Name)
		return err
	}
	if err = container.Delete(ctx, []containerd.DeleteOpts{}...); err != nil {
		log.Errorln("Failed to delete container", container.ID)
		return err
	}

	return nil
}
