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

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	k3d "github.com/rancher/k3d/v3/pkg/types"
	log "github.com/sirupsen/logrus"
)

// GetDefaultObjectLabelsFilter returns docker type filters created from k3d labels
func GetDefaultObjectLabelsFilter(clusterName string) filters.Args {
	filters := filters.NewArgs()
	for key, value := range k3d.DefaultObjectLabels {
		filters.Add("label", fmt.Sprintf("%s=%s", key, value))
	}
	filters.Add("label", fmt.Sprintf("%s=%s", k3d.LabelClusterName, clusterName))
	return filters
}

// CopyToNode copies a file from the local FS to the selected node
func (d Docker) CopyToNode(ctx context.Context, src string, dest string, node *k3d.Node) error {
	// create docker client
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Errorln("Failed to create docker client")
		return err
	}
	defer docker.Close()

	container, err := getNodeContainer(ctx, node)
	if err != nil {
		log.Errorf("Failed to find container for target node '%s'", node.Name)
		return err
	}

	// source: docker/cli/cli/command/container/cp
	srcInfo, err := archive.CopyInfoSourcePath(src, false)
	if err != nil {
		log.Errorln("Failed to copy info source path")
		return err
	}

	srcArchive, err := archive.TarResource(srcInfo)
	if err != nil {
		log.Errorln("Failed to create tar resource")
		return err
	}
	defer srcArchive.Close()

	destInfo := archive.CopyInfo{Path: dest}

	destStat, _ := docker.ContainerStatPath(ctx, container.ID, dest) // don't blame me, docker is also not doing anything if err != nil ¯\_(ツ)_/¯

	destInfo.Exists, destInfo.IsDir = true, destStat.Mode.IsDir()

	destDir, preparedArchive, err := archive.PrepareArchiveCopy(srcArchive, srcInfo, destInfo)
	if err != nil {
		log.Errorln("Failed to prepare archive")
		return err
	}
	defer preparedArchive.Close()

	return docker.CopyToContainer(ctx, container.ID, destDir, preparedArchive, types.CopyToContainerOptions{AllowOverwriteDirWithFile: false})
}
