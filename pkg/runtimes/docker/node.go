/*
Copyright © 2020-2021 The k3d Author(s)

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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	runtimeErr "github.com/rancher/k3d/v4/pkg/runtimes/errors"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	log "github.com/sirupsen/logrus"
)

// CreateNode creates a new container
func (d Docker) CreateNode(ctx context.Context, node *k3d.Node) error {

	// translate node spec to docker container specs
	dockerNode, err := TranslateNodeToContainer(node)
	if err != nil {
		log.Errorln("Failed to translate k3d node specification to docker container specifications")
		return err
	}

	// create node
	_, err = createContainer(ctx, dockerNode, node.Name)
	if err != nil {
		log.Errorf("Failed to create node '%s'", node.Name)
		return err
	}

	return nil
}

// DeleteNode deletes a node
func (d Docker) DeleteNode(ctx context.Context, nodeSpec *k3d.Node) error {
	log.Debugf("Deleting node %s ...", nodeSpec.Name)
	return removeContainer(ctx, nodeSpec.Name)
}

// GetNodesByLabel returns a list of existing nodes
func (d Docker) GetNodesByLabel(ctx context.Context, labels map[string]string) ([]*k3d.Node, error) {

	// (0) get containers
	containers, err := getContainersByLabel(ctx, labels)
	if err != nil {
		return nil, err
	}

	// (1) convert them to node structs
	nodes := []*k3d.Node{}
	for _, container := range containers {
		var node *k3d.Node
		var err error

		containerDetails, err := getContainerDetails(ctx, container.ID)
		if err != nil {
			log.Warnf("Failed to get details for container %s", container.Names[0])
			node, err = TranslateContainerToNode(&container)
			if err != nil {
				return nil, err
			}
		} else {
			node, err = TranslateContainerDetailsToNode(containerDetails)
			if err != nil {
				return nil, err
			}
		}
		nodes = append(nodes, node)
	}

	return nodes, nil

}

// StartNode starts an existing node
func (d Docker) StartNode(ctx context.Context, node *k3d.Node) error {
	// (0) create docker client
	docker, err := GetDockerClient()
	if err != nil {
		return fmt.Errorf("Failed to create docker client. %+v", err)
	}
	defer docker.Close()

	// get container which represents the node
	nodeContainer, err := getNodeContainer(ctx, node)
	if err != nil {
		log.Errorf("Failed to get container for node '%s'", node.Name)
		return err
	}

	// check if the container is actually managed by
	if v, ok := nodeContainer.Labels["app"]; !ok || v != "k3d" {
		return fmt.Errorf("Failed to determine if container '%s' is managed by k3d (needs label 'app=k3d')", nodeContainer.ID)
	}

	// actually start the container
	log.Infof("Starting Node '%s'", node.Name)
	if err := docker.ContainerStart(ctx, nodeContainer.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	// get container which represents the node
	nodeContainerJSON, err := docker.ContainerInspect(ctx, nodeContainer.ID)
	if err != nil {
		return fmt.Errorf("Failed to inspect container %s for node %s: %+v", node.Name, nodeContainer.ID, err)
	}

	node.Created = nodeContainerJSON.Created
	node.State.Running = nodeContainerJSON.State.Running
	node.State.Started = nodeContainerJSON.State.StartedAt

	return nil
}

// StopNode stops an existing node
func (d Docker) StopNode(ctx context.Context, node *k3d.Node) error {
	// (0) create docker client
	docker, err := GetDockerClient()
	if err != nil {
		return fmt.Errorf("Failed to create docker client. %+v", err)
	}
	defer docker.Close()

	// get container which represents the node
	nodeContainer, err := getNodeContainer(ctx, node)
	if err != nil {
		log.Errorf("Failed to get container for node '%s'", node.Name)
		return err
	}

	// check if the container is actually managed by
	if v, ok := nodeContainer.Labels["app"]; !ok || v != "k3d" {
		return fmt.Errorf("Failed to determine if container '%s' is managed by k3d (needs label 'app=k3d')", nodeContainer.ID)
	}

	// actually stop the container
	if err := docker.ContainerStop(ctx, nodeContainer.ID, nil); err != nil {
		return err
	}

	return nil
}

func getContainersByLabel(ctx context.Context, labels map[string]string) ([]types.Container, error) {
	// (0) create docker client
	docker, err := GetDockerClient()
	if err != nil {
		return nil, fmt.Errorf("Failed to create docker client. %+v", err)
	}
	defer docker.Close()

	// (1) list containers which have the default k3d labels attached
	filters := filters.NewArgs()
	for k, v := range k3d.DefaultRuntimeLabels {
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
		return nil, fmt.Errorf("Failed to list containers: %+v", err)
	}

	return containers, nil
}

// getContainer details returns the containerjson with more details
func getContainerDetails(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	// (0) create docker client
	docker, err := GetDockerClient()
	if err != nil {
		return types.ContainerJSON{}, fmt.Errorf("Failed to create docker client. %+v", err)
	}
	defer docker.Close()

	containerDetails, err := docker.ContainerInspect(ctx, containerID)
	if err != nil {
		log.Errorf("Failed to get details for container '%s'", containerID)
		return types.ContainerJSON{}, err
	}

	return containerDetails, nil

}

// GetNode tries to get a node container by its name
func (d Docker) GetNode(ctx context.Context, node *k3d.Node) (*k3d.Node, error) {
	container, err := getNodeContainer(ctx, node)
	if err != nil {
		return node, err
	}

	containerDetails, err := getContainerDetails(ctx, container.ID)
	if err != nil {
		return node, err
	}

	node, err = TranslateContainerDetailsToNode(containerDetails)
	if err != nil {
		log.Errorf("Failed to translate container '%s' to node object", containerDetails.Name)
		return node, err
	}

	return node, nil

}

// GetNodeStatus returns the status of a node (Running, Started, etc.)
func (d Docker) GetNodeStatus(ctx context.Context, node *k3d.Node) (bool, string, error) {

	stateString := ""
	running := false

	// get the container for the given node
	container, err := getNodeContainer(ctx, node)
	if err != nil {
		return running, stateString, err
	}

	// create docker client
	docker, err := GetDockerClient()
	if err != nil {
		log.Errorln("Failed to create docker client")
		return running, stateString, err
	}
	defer docker.Close()

	containerInspectResponse, err := docker.ContainerInspect(ctx, container.ID)
	if err != nil {
		return running, stateString, err
	}

	running = containerInspectResponse.ContainerJSONBase.State.Running
	stateString = containerInspectResponse.ContainerJSONBase.State.Status

	return running, stateString, nil
}

// NodeIsRunning tells the caller if a given node is in "running" state
func (d Docker) NodeIsRunning(ctx context.Context, node *k3d.Node) (bool, error) {
	isRunning, _, err := d.GetNodeStatus(ctx, node)
	if err != nil {
		return false, err
	}
	return isRunning, nil
}

// GetNodeLogs returns the logs from a given node
func (d Docker) GetNodeLogs(ctx context.Context, node *k3d.Node, since time.Time) (io.ReadCloser, error) {
	// get the container for the given node
	container, err := getNodeContainer(ctx, node)
	if err != nil {
		return nil, err
	}

	// create docker client
	docker, err := GetDockerClient()
	if err != nil {
		log.Errorln("Failed to create docker client")
		return nil, err
	}
	defer docker.Close()

	containerInspectResponse, err := docker.ContainerInspect(ctx, container.ID)
	if err != nil {
		log.Errorf("Failed to inspect node '%s'(ID %s)", node.Name, container.ID)
		return nil, err
	}

	if !containerInspectResponse.ContainerJSONBase.State.Running {
		return nil, fmt.Errorf("Node '%s' (container '%s') not running", node.Name, containerInspectResponse.ID)
	}

	sinceStr := ""
	if !since.IsZero() {
		sinceStr = since.Format("2006-01-02T15:04:05.999999999Z")
	}
	logreader, err := docker.ContainerLogs(ctx, container.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Since: sinceStr})
	if err != nil {
		log.Errorf("Failed to get logs from node '%s' (container '%s')", node.Name, container.ID)
		return nil, err
	}

	return logreader, nil
}

// ExecInNodeGetLogs executes a command inside a node and returns the logs to the caller, e.g. to parse them
func (d Docker) ExecInNodeGetLogs(ctx context.Context, node *k3d.Node, cmd []string) (*bufio.Reader, error) {
	resp, err := executeInNode(ctx, node, cmd)
	if err != nil {
		if resp != nil && resp.Reader != nil { // sometimes the exec process returns with a non-zero exit code, but we still have the logs we
			return resp.Reader, err
		}
		return nil, err
	}
	return resp.Reader, nil
}

// ExecInNode execs a command inside a node
func (d Docker) ExecInNode(ctx context.Context, node *k3d.Node, cmd []string) error {
	execConnection, err := executeInNode(ctx, node, cmd)
	if err != nil {
		if execConnection != nil && execConnection.Reader != nil {
			logs, err := ioutil.ReadAll(execConnection.Reader)
			if err != nil {
				log.Errorf("Failed to get logs from errored exec process in node '%s'", node.Name)
				return err
			}
			err = fmt.Errorf("%w: Logs from failed access process:\n%s", err, string(logs))
		}
	}
	return err
}

func executeInNode(ctx context.Context, node *k3d.Node, cmd []string) (*types.HijackedResponse, error) {

	log.Debugf("Executing command '%+v' in node '%s'", cmd, node.Name)

	// get the container for the given node
	container, err := getNodeContainer(ctx, node)
	if err != nil {
		return nil, err
	}

	// create docker client
	docker, err := GetDockerClient()
	if err != nil {
		log.Errorln("Failed to create docker client")
		return nil, err
	}
	defer docker.Close()

	// exec
	exec, err := docker.ContainerExecCreate(ctx, container.ID, types.ExecConfig{
		Privileged:   true,
		Tty:          true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          cmd,
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to create exec config for node '%s': %+v", node.Name, err)
	}

	execConnection, err := docker.ContainerExecAttach(ctx, exec.ID, types.ExecStartCheck{
		Tty: true,
	})
	if err != nil {
		log.Errorf("Failed to connect to exec process in node '%s'", node.Name)
		return nil, err
	}

	if err := docker.ContainerExecStart(ctx, exec.ID, types.ExecStartCheck{Tty: true}); err != nil {
		log.Errorf("Failed to start exec process in node '%s'", node.Name)
		return nil, err
	}

	for {
		// get info about exec process inside container
		execInfo, err := docker.ContainerExecInspect(ctx, exec.ID)
		if err != nil {
			log.Errorf("Failed to inspect exec process in node '%s'", node.Name)
			return &execConnection, err
		}

		// if still running, continue loop
		if execInfo.Running {
			log.Tracef("Exec process '%+v' still running in node '%s'.. sleeping for 1 second...", cmd, node.Name)
			time.Sleep(1 * time.Second)
			continue
		}

		// check exitcode
		if execInfo.ExitCode == 0 { // success
			log.Debugf("Exec process in node '%s' exited with '0'", node.Name)
			return &execConnection, nil
		}
		return &execConnection, fmt.Errorf("Exec process in node '%s' failed with exit code '%d'", node.Name, execInfo.ExitCode)
	}
}

// GetNodesInNetwork returns all the nodes connected to a given network
func (d Docker) GetNodesInNetwork(ctx context.Context, network string) ([]*k3d.Node, error) {
	// create docker client
	docker, err := GetDockerClient()
	if err != nil {
		log.Errorln("Failed to create docker client")
		return nil, err
	}
	defer docker.Close()

	net, err := GetNetwork(ctx, network)
	if err != nil {
		return nil, err
	}

	connectedNodes := []*k3d.Node{}

	// loop over list of containers connected to this cluster and transform them into nodes internally
	for cID := range net.Containers {
		containerDetails, err := getContainerDetails(ctx, cID)
		if err != nil {
			return nil, err
		}
		node, err := TranslateContainerDetailsToNode(containerDetails)
		if err != nil {
			if errors.Is(err, runtimeErr.ErrRuntimeContainerUnknown) {
				log.Tracef("GetNodesInNetwork: inspected non-k3d-managed container %s", containerDetails.Name)
				continue
			}
			return nil, err
		}
		connectedNodes = append(connectedNodes, node)
	}

	return connectedNodes, nil
}

func (d Docker) RenameNode(ctx context.Context, node *k3d.Node, newName string) error {
	// get the container for the given node
	container, err := getNodeContainer(ctx, node)
	if err != nil {
		return err
	}

	// create docker client
	docker, err := GetDockerClient()
	if err != nil {
		log.Errorln("Failed to create docker client")
		return err
	}
	defer docker.Close()

	return docker.ContainerRename(ctx, container.ID, newName)
}
