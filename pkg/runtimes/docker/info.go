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
	"strings"

	runtimeTypes "github.com/k3d-io/k3d/v5/pkg/runtimes/types"
)

func (d Docker) Info() (*runtimeTypes.RuntimeInfo, error) {
	// create docker client
	docker, err := GetDockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	defer docker.Close()

	info, err := docker.Info(context.Background())
	if err != nil {
		return nil, fmt.Errorf("docker failed to provide info output: %w", err)
	}

	runtimeInfo := runtimeTypes.RuntimeInfo{
		Name:          d.ID(),
		Endpoint:      d.GetRuntimePath(),
		Version:       info.ServerVersion,
		OS:            info.OperatingSystem,
		OSType:        info.OSType,
		Arch:          info.Architecture,
		CgroupVersion: info.CgroupVersion,
		CgroupDriver:  info.CgroupDriver,
		Filesystem:    "UNKNOWN",
		InfoName:      info.Name,
	}

	// Get the backing filesystem for the storage driver
	// This is not embedded nicely in a struct or map, so we have to do some string inspection
	for i := range info.DriverStatus {
		for j := range info.DriverStatus[i] {
			if strings.Contains(info.DriverStatus[i][j], "Backing Filesystem") {
				if len(info.DriverStatus[i]) >= j+2 {
					runtimeInfo.Filesystem = info.DriverStatus[i][j+1]
				}
			}
		}
	}

	return &runtimeInfo, nil
}
