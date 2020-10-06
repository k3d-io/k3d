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
	"strings"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	k3d "github.com/rancher/k3d/v3/pkg/types"
	log "github.com/sirupsen/logrus"
)

// TranslateNodeToContainer translates a k3d node specification to a docker container representation
func TranslateNodeToContainer(node *k3d.Node) (*NodeInDocker, error) {

	/* initialize everything that we need */
	containerConfig := docker.Config{}
	hostConfig := docker.HostConfig{
		Init:       &[]bool{true}[0],
		ExtraHosts: node.ExtraHosts,
	}
	networkingConfig := network.NetworkingConfig{}

	/* Name & Image */
	containerConfig.Hostname = node.Name
	containerConfig.Image = node.Image

	/* Command & Arguments */
	containerConfig.Cmd = []string{}

	containerConfig.Cmd = append(containerConfig.Cmd, node.Cmd...)  // contains k3s command and role-specific required flags/args
	containerConfig.Cmd = append(containerConfig.Cmd, node.Args...) // extra flags/args

	/* Environment Variables */
	containerConfig.Env = node.Env

	/* Labels */
	containerConfig.Labels = node.Labels // has to include the role

	/* Auto-Restart */
	if node.Restart {
		hostConfig.RestartPolicy = docker.RestartPolicy{
			Name: "unless-stopped",
		}
	}

	/* Tmpfs Mounts */
	hostConfig.Tmpfs = make(map[string]string)
	for _, mnt := range k3d.DefaultTmpfsMounts {
		hostConfig.Tmpfs[mnt] = ""
	}

	/* They have to run in privileged mode */
	// TODO: can we replace this by a reduced set of capabilities?
	hostConfig.Privileged = true

	/* Volumes */
	log.Debugf("Volumes: %+v", node.Volumes)
	hostConfig.Binds = node.Volumes
	// containerConfig.Volumes = map[string]struct{}{} // TODO: do we need this? We only used binds before

	/* Ports */
	exposedPorts, portBindings, err := nat.ParsePortSpecs(node.Ports)
	if err != nil {
		log.Errorf("Failed to parse port specs '%v'", node.Ports)
		return nil, err
	}
	containerConfig.ExposedPorts = exposedPorts
	hostConfig.PortBindings = portBindings
	/* Network */
	networkingConfig.EndpointsConfig = map[string]*network.EndpointSettings{
		node.Network: {},
	}
	netInfo, err := GetNetwork(context.Background(), node.Network)
	if err != nil {
		log.Warnln("Failed to get network information")
		log.Warnln(err)
	} else if netInfo.Driver == "host" {
		hostConfig.NetworkMode = "host"
	}

	return &NodeInDocker{
		ContainerConfig:  containerConfig,
		HostConfig:       hostConfig,
		NetworkingConfig: networkingConfig,
	}, nil
}

// TranslateContainerToNode translates a docker container object into a k3d node representation
func TranslateContainerToNode(cont *types.Container) (*k3d.Node, error) {
	node := &k3d.Node{
		Name:   strings.TrimPrefix(cont.Names[0], "/"), // container name with leading '/' cut off
		Image:  cont.Image,
		Labels: cont.Labels,
		Role:   k3d.NodeRoles[cont.Labels[k3d.LabelRole]],
		// TODO: all the rest
	}
	return node, nil
}

// TranslateContainerDetailsToNode translates a docker containerJSON object into a k3d node representation
func TranslateContainerDetailsToNode(containerDetails types.ContainerJSON) (*k3d.Node, error) {

	// translate portMap to string representation
	ports := []string{}
	for containerPort, portBindingList := range containerDetails.HostConfig.PortBindings {
		for _, hostInfo := range portBindingList {
			ports = append(ports, fmt.Sprintf("%s:%s:%s", hostInfo.HostIP, hostInfo.HostPort, containerPort))
		}
	}

	// restart -> we only set 'unless-stopped' upon cluster creation
	restart := false
	if containerDetails.HostConfig.RestartPolicy.IsAlways() || containerDetails.HostConfig.RestartPolicy.IsUnlessStopped() {
		restart = true
	}

	// get the clusterNetwork
	clusterNetwork := ""
	for networkName := range containerDetails.NetworkSettings.Networks {
		if strings.HasPrefix(networkName, fmt.Sprintf("%s-%s", k3d.DefaultObjectNamePrefix, containerDetails.Config.Labels[k3d.LabelClusterName])) { // FIXME: catch error if label 'k3d.cluster' does not exist, but this should also never be the case
			clusterNetwork = networkName
		}
	}

	// serverOpts
	serverOpts := k3d.ServerOpts{IsInit: false}
	for k, v := range containerDetails.Config.Labels {
		if k == k3d.LabelServerAPIHostIP {
			serverOpts.ExposeAPI.HostIP = v
		} else if k == k3d.LabelServerAPIHost {
			serverOpts.ExposeAPI.Host = v
		} else if k == k3d.LabelServerAPIPort {
			serverOpts.ExposeAPI.Port = v
		}
	}

	// env vars: only copy K3S_* and K3D_* // FIXME: should we really do this? Might be unexpected, if user has e.g. HTTP_PROXY vars
	env := []string{}
	for _, envVar := range containerDetails.Config.Env {
		if strings.HasPrefix(envVar, "K3D_") || strings.HasPrefix(envVar, "K3S_") {
			env = append(env, envVar)
		}
	}

	// labels: only copy k3d.* labels
	labels := map[string]string{}
	for k, v := range containerDetails.Config.Labels {
		if strings.HasPrefix(k, "k3d") {
			labels[k] = v
		}
	}

	// status
	nodeState := k3d.NodeState{
		Running: containerDetails.ContainerJSONBase.State.Running,
		Status:  containerDetails.ContainerJSONBase.State.Status,
	}

	node := &k3d.Node{
		Name:       strings.TrimPrefix(containerDetails.Name, "/"), // container name with leading '/' cut off
		Role:       k3d.NodeRoles[containerDetails.Config.Labels[k3d.LabelRole]],
		Image:      containerDetails.Image,
		Volumes:    containerDetails.HostConfig.Binds,
		Env:        env,
		Cmd:        containerDetails.Config.Cmd,
		Args:       []string{}, // empty, since Cmd already contains flags
		Ports:      ports,
		Restart:    restart,
		Labels:     labels,
		Network:    clusterNetwork,
		ServerOpts: serverOpts,
		AgentOpts:  k3d.AgentOpts{},
		State:      nodeState,
	}
	return node, nil
}
