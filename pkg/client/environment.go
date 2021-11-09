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
package client

import (
	"context"
	"fmt"

	l "github.com/rancher/k3d/v5/pkg/logger"
	"github.com/rancher/k3d/v5/pkg/runtimes"

	k3d "github.com/rancher/k3d/v5/pkg/types"
)

func GatherEnvironmentInfo(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster) (*k3d.EnvironmentInfo, error) {

	envInfo := &k3d.EnvironmentInfo{}

	rtimeInfo, err := runtime.Info()
	if err != nil {
		return nil, err
	}
	envInfo.RuntimeInfo = *rtimeInfo

	l.Log().Infof("Using the k3d-tools node to gather environment information")
	toolsNode, err := EnsureToolsNode(ctx, runtime, cluster)
	if err != nil {
		return nil, err
	}
	defer func() {
		go NodeDelete(ctx, runtime, toolsNode, k3d.NodeDeleteOpts{SkipLBUpdate: true})
	}()

	if cluster.Network.Name != "host" {
		hostIP, err := GetHostIP(ctx, runtime, cluster)
		if err != nil {
			return envInfo, fmt.Errorf("failed to get host IP: %w", err)
		}

		envInfo.HostGateway = hostIP
	}

	return envInfo, nil

}
