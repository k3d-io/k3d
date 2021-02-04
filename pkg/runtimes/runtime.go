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
package runtimes

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/rancher/k3d/v4/pkg/runtimes/containerd"
	"github.com/rancher/k3d/v4/pkg/runtimes/docker"
	k3d "github.com/rancher/k3d/v4/pkg/types"
)

// SelectedRuntime is a runtime (pun intended) variable determining the selected runtime
var SelectedRuntime Runtime = docker.Docker{}

// Docker docker
var Docker = docker.Docker{}

// Containerd containerd
var Containerd = containerd.Containerd{}

// Runtimes defines a map of implemented k3d runtimes
var Runtimes = map[string]Runtime{
	"docker":     docker.Docker{},
	"containerd": containerd.Containerd{},
}

// Runtime defines an interface that can be implemented for various container runtime environments (docker, containerd, etc.)
type Runtime interface {
	ID() string
	GetHost() string
	CreateNode(context.Context, *k3d.Node) error
	DeleteNode(context.Context, *k3d.Node) error
	GetNodesByLabel(context.Context, map[string]string) ([]*k3d.Node, error)
	GetNode(context.Context, *k3d.Node) (*k3d.Node, error)
	GetNodeStatus(context.Context, *k3d.Node) (bool, string, error)
	GetNodesInNetwork(context.Context, string) ([]*k3d.Node, error)
	CreateNetworkIfNotPresent(context.Context, string) (string, bool, error) // @return NETWORK_NAME, EXISTS, ERROR
	GetKubeconfig(context.Context, *k3d.Node) (io.ReadCloser, error)
	DeleteNetwork(context.Context, string) error
	StartNode(context.Context, *k3d.Node) error // starts an existing container
	StopNode(context.Context, *k3d.Node) error
	CreateVolume(context.Context, string, map[string]string) error
	DeleteVolume(context.Context, string) error
	GetVolume(string) (string, error)
	GetRuntimePath() string // returns e.g. '/var/run/docker.sock' for a default docker setup
	ExecInNode(context.Context, *k3d.Node, []string) error
	ExecInNodeGetLogs(context.Context, *k3d.Node, []string) (*bufio.Reader, error)
	GetNodeLogs(context.Context, *k3d.Node, time.Time) (io.ReadCloser, error)
	GetImages(context.Context) ([]string, error)
	CopyToNode(context.Context, string, string, *k3d.Node) error  // @param context, source, destination, node
	WriteToNode(context.Context, []byte, string, *k3d.Node) error // @param context, content, destination, node
	GetHostIP(context.Context, string) (net.IP, error)
	ConnectNodeToNetwork(context.Context, *k3d.Node, string) error      // @param context, node, network name
	DisconnectNodeFromNetwork(context.Context, *k3d.Node, string) error // @param context, node, network name
}

// GetRuntime checks, if a given name is represented by an implemented k3d runtime and returns it
func GetRuntime(rt string) (Runtime, error) {
	if runtime, ok := Runtimes[rt]; ok {
		return runtime, nil
	}
	return nil, fmt.Errorf("Runtime '%s' not supported", rt)
}
