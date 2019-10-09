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
	docker "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	k3d "github.com/rancher/k3d/pkg/types"
)

func TranslateNodeToContainer(node *k3d.Node) (docker.Config, error) {

	container := docker.Config{}
	container.Hostname = node.Name
	container.Image = node.Image
	container.Labels = node.Labels // has to include the role
	container.Env = []string{}     // TODO:
	container.Cmd = []string{}     // TODO: dependent on role and extra args
	hostConfig := docker.HostConfig{}

	/* Auto-Restart */
	if node.Restart {
		hostConfig.RestartPolicy = docker.RestartPolicy{
			Name: "unless-stopped",
		}
	}

	// TODO: do we need this or can the default be a map with empty values already?
	hostConfig.Tmpfs = make(map[string]string)
	for _, mnt := range k3d.DefaultTmpfsMounts {
		hostConfig.Tmpfs[mnt] = ""
	}

	hostConfig.Privileged = true

	/* Volumes */
	// TODO: image volume
	hostConfig.Binds = []string{}
	container.Volumes = map[string]struct{}{} // TODO: which one do we use?

	/* Ports */
	container.ExposedPorts = nat.PortSet{}  // TODO:
	hostConfig.PortBindings = nat.PortMap{} // TODO: this and exposedPorts required?

	/* Network */
	networkingConfig := &network.NetworkingConfig{}
	networkingConfig.EndpointsConfig = map[string]*network.EndpointSettings{
		"<network-name>": { // TODO: fill
			Aliases: []string{"<container-alias>"}, // TODO: fill
		},
	}

	return container, nil
}
