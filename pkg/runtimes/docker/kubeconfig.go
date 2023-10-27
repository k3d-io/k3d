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
	"io"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

// GetKubeconfig grabs the kubeconfig from inside a k3d node
func (d Docker) GetKubeconfig(ctx context.Context, node *k3d.Node) (io.ReadCloser, error) {
	docker, err := GetDockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	defer docker.Close()

	container, err := getNodeContainer(ctx, node)
	if err != nil {
		return nil, fmt.Errorf("docker failed to get container for node '%s': %w", node.Name, err)
	}

	l.Log().Tracef("Container Details: %+v", container)

	reader, _, err := docker.CopyFromContainer(ctx, container.ID, "/output/kubeconfig.yaml")
	if err != nil {
		return nil, fmt.Errorf("docker failed to copy path '/output/kubeconfig.yaml' from container '%s': %w", container.ID, err)
	}

	return reader, nil
}
