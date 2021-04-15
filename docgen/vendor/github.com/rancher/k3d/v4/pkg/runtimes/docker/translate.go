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
	runtimeErr "github.com/rancher/k3d/v4/pkg/runtimes/errors"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	log "github.com/sirupsen/logrus"

	dockercliopts "github.com/docker/cli/opts"
	dockerunits "github.com/docker/go-units"
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

	if node.GPURequest != "" {
		gpuopts := dockercliopts.GpuOpts{}
		if err := gpuopts.Set(node.GPURequest); err != nil {
			return nil, fmt.Errorf("Failed to set GPU Request: %+v", err)
		}
		hostConfig.DeviceRequests = gpuopts.Value()
	}

	// memory limits
	// fake meminfo is mounted to hostConfig.Binds
	if node.Memory != "" {
		memory, err := dockerunits.RAMInBytes(node.Memory)
		if err != nil {
			return nil, fmt.Errorf("Failed to set memory limit: %+v", err)
		}
		hostConfig.Memory = memory
	}

	/* They have to run in privileged mode */
	// TODO: can we replace this by a reduced set of capabilities?
	hostConfig.Privileged = true

	/* Volumes */
	hostConfig.Binds = node.Volumes
	// containerConfig.Volumes = map[string]struct{}{} // TODO: do we need this? We only used binds before

	/* Ports */
	exposedPorts := nat.PortSet{}
	for ep := range node.Ports {
		if _, exists := exposedPorts[ep]; !exists {
			exposedPorts[ep] = struct{}{}
		}
	}
	containerConfig.ExposedPorts = exposedPorts
	hostConfig.PortBindings = node.Ports

	/* Network */
	endpointsConfig := map[string]*network.EndpointSettings{}
	for _, net := range node.Networks {
		epSettings := &network.EndpointSettings{}
		endpointsConfig[net] = epSettings
	}

	networkingConfig.EndpointsConfig = endpointsConfig

	/* Static IP */
	if node.IP.IP != "" && node.IP.Static {
		epconf := networkingConfig.EndpointsConfig[node.Networks[0]]
		if epconf.IPAMConfig == nil {
			epconf.IPAMConfig = &network.EndpointIPAMConfig{}
		}
		epconf.IPAMConfig.IPv4Address = node.IP.IP
	}

	if len(node.Networks) > 0 {
		netInfo, err := GetNetwork(context.Background(), node.Networks[0]) // FIXME: only considering first network here, as that's the one k3d creates for a cluster
		if err != nil {
			log.Warnln("Failed to get network information")
			log.Warnln(err)
		} else if netInfo.Driver == "host" {
			hostConfig.NetworkMode = "host"
		}
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

	// first, make sure, that it's actually a k3d managed container by checking if it has all the default labels
	for k, v := range k3d.DefaultObjectLabels {
		log.Tracef("TranslateContainerDetailsToNode: Checking for default object label %s=%s on container %s", k, v, containerDetails.Name)
		found := false
		for lk, lv := range containerDetails.Config.Labels {
			if lk == k && lv == v {
				found = true
				break
			}
		}
		if !found {
			log.Debugf("Container %s is missing default label %s=%s in label set %+v", containerDetails.Name, k, v, containerDetails.Config.Labels)
			return nil, runtimeErr.ErrRuntimeContainerUnknown
		}
	}

	// restart -> we only set 'unless-stopped' upon cluster creation
	restart := false
	if containerDetails.HostConfig.RestartPolicy.IsAlways() || containerDetails.HostConfig.RestartPolicy.IsUnlessStopped() {
		restart = true
	}

	// get networks and ensure that the cluster network is first in list
	orderedNetworks := []string{}
	otherNetworks := []string{}
	for networkName := range containerDetails.NetworkSettings.Networks {
		if strings.HasPrefix(networkName, fmt.Sprintf("%s-%s", k3d.DefaultObjectNamePrefix, containerDetails.Config.Labels[k3d.LabelClusterName])) { // FIXME: catch error if label 'k3d.cluster' does not exist, but this should also never be the case
			orderedNetworks = append(orderedNetworks, networkName)
			continue
		}
		otherNetworks = append(otherNetworks, networkName)
	}
	orderedNetworks = append(orderedNetworks, otherNetworks...)

	/**
	 * ServerOpts
	 */

	// IsInit
	serverOpts := k3d.ServerOpts{IsInit: false}
	clusterInitFlagSet := false
	for _, arg := range containerDetails.Args {
		if strings.Contains(arg, "--cluster-init") {
			clusterInitFlagSet = true
			break
		}
	}
	if serverIsInitLabel, ok := containerDetails.Config.Labels[k3d.LabelServerIsInit]; ok {
		if serverIsInitLabel == "true" {
			if !clusterInitFlagSet {
				log.Errorf("Container %s has label %s=true, but the args do not contain the --cluster-init flag", containerDetails.Name, k3d.LabelServerIsInit)
			} else {
				serverOpts.IsInit = true
			}
		}
	}

	// Kube API
	serverOpts.KubeAPI = &k3d.ExposureOpts{}
	for k, v := range containerDetails.Config.Labels {
		if k == k3d.LabelServerAPIHostIP {
			serverOpts.KubeAPI.Binding.HostIP = v
		} else if k == k3d.LabelServerAPIHost {
			serverOpts.KubeAPI.Host = v
		} else if k == k3d.LabelServerAPIPort {
			serverOpts.KubeAPI.Binding.HostPort = v
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

	// memory limit
	memoryStr := dockerunits.HumanSize(float64(containerDetails.HostConfig.Memory))
	// no-limit is returned as 0B, filter this out
	if memoryStr == "0B" {
		memoryStr = ""
	}

	node := &k3d.Node{
		Name:       strings.TrimPrefix(containerDetails.Name, "/"), // container name with leading '/' cut off
		Role:       k3d.NodeRoles[containerDetails.Config.Labels[k3d.LabelRole]],
		Image:      containerDetails.Image,
		Volumes:    containerDetails.HostConfig.Binds,
		Env:        env,
		Cmd:        containerDetails.Config.Cmd,
		Args:       []string{}, // empty, since Cmd already contains flags
		Ports:      containerDetails.HostConfig.PortBindings,
		Restart:    restart,
		Created:    containerDetails.Created,
		Labels:     labels,
		Networks:   orderedNetworks,
		ServerOpts: serverOpts,
		AgentOpts:  k3d.AgentOpts{},
		State:      nodeState,
		Memory:     memoryStr,
	}
	return node, nil
}
