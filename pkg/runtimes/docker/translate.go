/*
Copyright © 2020-2023 The k3d Author(s)

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
	"net/netip"
	"os"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	runtimeErr "github.com/k3d-io/k3d/v5/pkg/runtimes/errors"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"

	dockercliopts "github.com/docker/cli/opts"
	dockerunits "github.com/docker/go-units"
)

// TranslateNodeToContainer translates a k3d node specification to a docker container representation
func TranslateNodeToContainer(node *k3d.Node) (*NodeInDocker, error) {
	init := true
	if disableInit, err := strconv.ParseBool(os.Getenv(k3d.K3dEnvDebugDisableDockerInit)); err == nil && disableInit {
		l.Log().Traceln("docker-init disabled for all containers")
		init = false
	}

	/* initialize everything that we need */
	containerConfig := docker.Config{}
	hostConfig := docker.HostConfig{
		Init:       &init,
		ExtraHosts: node.ExtraHosts,
		// Explicitly require bridge networking. Podman incorrectly uses
		// slirp4netns when running rootless, therefore for rootless podman to
		// work, this must be set.
		NetworkMode: "bridge",
	}
	networkingConfig := network.NetworkingConfig{}

	/* Name & Image */
	containerConfig.Hostname = node.Name
	containerConfig.Image = node.Image

	/* Command & Arguments */
	if node.K3dEntrypoint {
		if node.Role == k3d.AgentRole || node.Role == k3d.ServerRole {
			containerConfig.Entrypoint = []string{
				"/bin/k3d-entrypoint.sh",
			}
		}
	}

	containerConfig.Cmd = []string{}

	containerConfig.Cmd = append(containerConfig.Cmd, node.Cmd...)  // contains k3s command and role-specific required flags/args
	containerConfig.Cmd = append(containerConfig.Cmd, node.Args...) // extra flags/args

	/* Environment Variables */
	containerConfig.Env = node.Env

	/* Labels */
	containerConfig.Labels = node.RuntimeLabels // has to include the role

	/* Ulimits */
	if len(node.RuntimeUlimits) > 0 {
		hostConfig.Ulimits = node.RuntimeUlimits
	}

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

	// Privileged containers require userns=host when Docker has userns-remap enabled
	hostConfig.UsernsMode = "host"

	if node.HostPidMode {
		hostConfig.PidMode = "host"
	}

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
	if node.IP.IP.IsValid() && node.IP.Static {
		epconf := networkingConfig.EndpointsConfig[node.Networks[0]]
		if epconf.IPAMConfig == nil {
			epconf.IPAMConfig = &network.EndpointIPAMConfig{}
		}
		epconf.IPAMConfig.IPv4Address = node.IP.IP.String()
	}

	if len(node.Networks) > 0 {
		netInfo, err := GetNetwork(context.Background(), node.Networks[0]) // FIXME: only considering first network here, as that's the one k3d creates for a cluster
		if err != nil {
			l.Log().Warnf("Failed to get network information: %v", err)
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
		Name:          strings.TrimPrefix(cont.Names[0], "/"), // container name with leading '/' cut off
		Image:         cont.Image,
		RuntimeLabels: cont.Labels,
		Role:          k3d.NodeRoles[cont.Labels[k3d.LabelRole]],
		// TODO: all the rest
	}
	return node, nil
}

// TranslateContainerDetailsToNode translates a docker containerJSON object into a k3d node representation
func TranslateContainerDetailsToNode(containerDetails types.ContainerJSON) (*k3d.Node, error) {
	// first, make sure, that it's actually a k3d managed container by checking if it has all the default labels
	for k, v := range k3d.DefaultRuntimeLabels {
		l.Log().Tracef("TranslateContainerDetailsToNode: Checking for default object label %s=%s on container %s", k, v, containerDetails.Name)
		found := false
		for lk, lv := range containerDetails.Config.Labels {
			if lk == k && lv == v {
				found = true
				break
			}
		}
		if !found {
			l.Log().Debugf("Container %s is missing default label %s=%s in label set %+v", containerDetails.Name, k, v, containerDetails.Config.Labels)
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
				l.Log().Errorf("Container %s has label %s=true, but the args do not contain the --cluster-init flag", containerDetails.Name, k3d.LabelServerIsInit)
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

	// IP
	var nodeIP k3d.NodeIP
	var clusterNet *network.EndpointSettings
	if netLabel, ok := labels[k3d.LabelNetwork]; ok {
		for netName, net := range containerDetails.NetworkSettings.Networks {
			if netName == netLabel {
				clusterNet = net
			}
		}
	} else {
		l.Log().Debugf("no netlabel present on container %s", containerDetails.Name)
	}
	if clusterNet != nil && labels[k3d.LabelNetwork] != "host" {
		parsedIP, err := netip.ParseAddr(clusterNet.IPAddress)
		if err != nil {
			if nodeState.Running && nodeState.Status != "restarting" { // if the container is not running or currently restarting, it won't have an IP, so we don't error in that case
				return nil, fmt.Errorf("failed to parse IP '%s' for container '%s': %s\nStatus: %v\n%+v", clusterNet.IPAddress, containerDetails.Name, err, nodeState.Status, containerDetails.NetworkSettings)
			} else {
				l.Log().Tracef("failed to parse IP '%s' for container '%s', likely because it's not running (or restarting): %v", clusterNet.IPAddress, containerDetails.Name, err)
			}
		}
		isStaticIP := false
		if staticIPLabel, ok := labels[k3d.LabelNodeStaticIP]; ok && staticIPLabel != "" {
			isStaticIP = true
		}
		if parsedIP.IsValid() {
			nodeIP = k3d.NodeIP{
				IP:     parsedIP,
				Static: isStaticIP,
			}
		}
	} else {
		l.Log().Debugf("failed to get IP for container %s as we couldn't find the cluster network", containerDetails.Name)
	}

	node := &k3d.Node{
		Name:          strings.TrimPrefix(containerDetails.Name, "/"), // container name with leading '/' cut off
		Role:          k3d.NodeRoles[containerDetails.Config.Labels[k3d.LabelRole]],
		Image:         containerDetails.Image,
		Volumes:       containerDetails.HostConfig.Binds,
		Env:           containerDetails.Config.Env,
		Cmd:           containerDetails.Config.Cmd,
		Args:          []string{}, // empty, since Cmd already contains flags
		Ports:         containerDetails.HostConfig.PortBindings,
		Restart:       restart,
		Created:       containerDetails.Created,
		RuntimeLabels: labels,
		Networks:      orderedNetworks,
		ServerOpts:    serverOpts,
		AgentOpts:     k3d.AgentOpts{},
		State:         nodeState,
		Memory:        memoryStr,
		IP:            nodeIP, // only valid for the cluster network
	}
	return node, nil
}
