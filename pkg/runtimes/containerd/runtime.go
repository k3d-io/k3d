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
	"context"

	"github.com/containerd/containerd"
	clog "github.com/containerd/containerd/log"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/nerdctl/pkg/infoutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/rancher/k3d/v5/pkg/runtimes/types"

	l "github.com/rancher/k3d/v5/pkg/logger"
)

func init() {
	// Set log level of the default global logger in containerd to WARN to avoid log spamming
	clog.L.Logger.SetLevel(logrus.WarnLevel)
}

// Containerd implements the k3d Runtime interface
type Containerd struct{}

// RuntimeIdentifierContainerd specifies the constant string identifying the containerd runtime in k3d
const RuntimeIdentifierContainerd string = "containerd"

// default values for containerd connections
const (
	ContainerdInDockerSock string = "/run/containerd/containerd.sock"
	DockerNamespace        string = "moby"
)

/*
 * Base Functions
 */

// GetClient returns a containerd API client
func GetClient(ctx context.Context) (*containerd.Client, context.Context, error) {

	ctx = namespaces.WithNamespace(ctx, DockerNamespace) // FIXME: namespace shouldn't be hardcoded here

	client, err := containerd.New(ContainerdInDockerSock)
	if err != nil {
		return nil, ctx, errors.Wrap(err, ErrGetContainerdClient.Error())
	}
	return client, ctx, nil
}

/*
 * Interface Functions
 */

// ID returns the runtime identifier
func (cd Containerd) ID() string {
	return RuntimeIdentifierContainerd
}

// Info returns information about the underlying runtime (server side info)
func (cd Containerd) Info() (*types.RuntimeInfo, error) {
	c, ctx, err := GetClient(context.Background())
	if err != nil {
		return nil, err
	}

	i, err := infoutil.Info(ctx, c, "", "") // FIXME: pass in snapshotter, etc.?
	if err != nil {
		return nil, err
	}

	info := &types.RuntimeInfo{
		Name:          cd.ID(),
		Endpoint:      c.Conn().Target(),
		Arch:          i.Architecture,
		OSType:        i.OSType,
		OS:            i.OperatingSystem,
		CgroupVersion: i.CgroupVersion,
		CgroupDriver:  i.CgroupDriver,
		Version:       i.ServerVersion,
		Filesystem:    "UNKNOWN",
	}

	l.Log().Tracef("INFO: %#v", i)

	return info, nil

}
