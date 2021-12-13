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
package containerd

import (
	"bufio"
	"context"
	"io"
	"net"
	"os"
	"time"

	runtimeTypes "github.com/rancher/k3d/v5/pkg/runtimes/types"
	k3d "github.com/rancher/k3d/v5/pkg/types"
)

func (cd Containerd) GetHost() string {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) CreateNode(_ context.Context, _ *k3d.Node) error {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) DeleteNode(_ context.Context, _ *k3d.Node) error {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) RenameNode(_ context.Context, _ *k3d.Node, _ string) error {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) GetNodesByLabel(_ context.Context, _ map[string]string) ([]*k3d.Node, error) {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) GetNode(_ context.Context, _ *k3d.Node) (*k3d.Node, error) {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) GetNodeStatus(_ context.Context, _ *k3d.Node) (bool, string, error) {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) GetNodesInNetwork(_ context.Context, _ string) ([]*k3d.Node, error) {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) CreateNetworkIfNotPresent(_ context.Context, _ *k3d.ClusterNetwork) (*k3d.ClusterNetwork, bool, error) {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) GetKubeconfig(_ context.Context, _ *k3d.Node) (io.ReadCloser, error) {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) DeleteNetwork(_ context.Context, _ string) error {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) StartNode(_ context.Context, _ *k3d.Node) error {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) StopNode(_ context.Context, _ *k3d.Node) error {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) CreateVolume(_ context.Context, _ string, _ map[string]string) error {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) DeleteVolume(_ context.Context, _ string) error {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) GetVolume(_ string) (string, error) {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) GetImageStream(_ context.Context, _ []string) (io.ReadCloser, error) {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) GetRuntimePath() string {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) ExecInNode(_ context.Context, _ *k3d.Node, _ []string) error {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) ExecInNodeWithStdin(_ context.Context, _ *k3d.Node, _ []string, _ io.ReadCloser) error {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) ExecInNodeGetLogs(_ context.Context, _ *k3d.Node, _ []string) (*bufio.Reader, error) {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) GetNodeLogs(_ context.Context, _ *k3d.Node, _ time.Time, _ *runtimeTypes.NodeLogsOpts) (io.ReadCloser, error) {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) GetImages(_ context.Context) ([]string, error) {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) CopyToNode(_ context.Context, _ string, _ string, _ *k3d.Node) error {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) WriteToNode(_ context.Context, _ []byte, _ string, _ os.FileMode, _ *k3d.Node) error {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) ReadFromNode(_ context.Context, _ string, _ *k3d.Node) (io.ReadCloser, error) {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) GetHostIP(_ context.Context, _ string) (net.IP, error) {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) ConnectNodeToNetwork(_ context.Context, _ *k3d.Node, _ string) error {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) DisconnectNodeFromNetwork(_ context.Context, _ *k3d.Node, _ string) error {
	panic("not implemented") // TODO: Implement
}

func (cd Containerd) GetNetwork(_ context.Context, _ *k3d.ClusterNetwork) (*k3d.ClusterNetwork, error) {
	panic("not implemented") // TODO: Implement
}
