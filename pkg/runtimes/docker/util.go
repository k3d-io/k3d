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
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/pkg/errors"
	runtimeErrors "github.com/rancher/k3d/v4/pkg/runtimes/errors"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	log "github.com/sirupsen/logrus"
)

// GetDefaultObjectLabelsFilter returns docker type filters created from k3d labels
func GetDefaultObjectLabelsFilter(clusterName string) filters.Args {
	filters := filters.NewArgs()
	for key, value := range k3d.DefaultRuntimeLabels {
		filters.Add("label", fmt.Sprintf("%s=%s", key, value))
	}
	filters.Add("label", fmt.Sprintf("%s=%s", k3d.LabelClusterName, clusterName))
	return filters
}

// CopyToNode copies a file from the local FS to the selected node
func (d Docker) CopyToNode(ctx context.Context, src string, dest string, node *k3d.Node) error {
	// create docker client
	docker, err := GetDockerClient()
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

// WriteToNode writes a byte array to the selected node
func (d Docker) WriteToNode(ctx context.Context, content []byte, dest string, mode os.FileMode, node *k3d.Node) error {

	nodeContainer, err := getNodeContainer(ctx, node)
	if err != nil {
		return fmt.Errorf("Failed to find container for node '%s': %+v", node.Name, err)
	}

	// create docker client
	docker, err := GetDockerClient()
	if err != nil {
		log.Errorln("Failed to create docker client")
		return err
	}
	defer docker.Close()

	buf := new(bytes.Buffer)
	tarWriter := tar.NewWriter(buf)
	defer tarWriter.Close()
	tarHeader := &tar.Header{
		Name: dest,
		Mode: int64(mode),
		Size: int64(len(content)),
	}

	if err := tarWriter.WriteHeader(tarHeader); err != nil {
		return fmt.Errorf("Failed to write tar header: %+v", err)
	}

	if _, err := tarWriter.Write(content); err != nil {
		return fmt.Errorf("Failed to write tar content: %+v", err)
	}

	if err := tarWriter.Close(); err != nil {
		log.Debugf("Failed to close tar writer: %+v", err)
	}

	tarBytes := bytes.NewReader(buf.Bytes())
	if err := docker.CopyToContainer(ctx, nodeContainer.ID, "/", tarBytes, types.CopyToContainerOptions{AllowOverwriteDirWithFile: true}); err != nil {
		return fmt.Errorf("Failed to copy content to container '%s': %+v", nodeContainer.ID, err)
	}

	return nil
}

// ReadFromNode reads from a given filepath inside the node container
func (d Docker) ReadFromNode(ctx context.Context, path string, node *k3d.Node) (io.ReadCloser, error) {
	log.Tracef("Reading path %s from node %s...", path, node.Name)
	nodeContainer, err := getNodeContainer(ctx, node)
	if err != nil {
		return nil, fmt.Errorf("Failed to find container for node '%s': %+v", node.Name, err)
	}

	docker, err := GetDockerClient()
	if err != nil {
		return nil, err
	}

	reader, _, err := docker.CopyFromContainer(ctx, nodeContainer.ID, path)
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil, errors.Wrap(runtimeErrors.ErrRuntimeFileNotFound, err.Error())
		}
		return nil, err
	}

	return reader, err
}

// GetDockerClient returns a docker client
func GetDockerClient() (*client.Client, error) {
	var err error
	var cli *client.Client

	dockerHost := os.Getenv("DOCKER_HOST")

	if strings.HasPrefix(dockerHost, "ssh://") {
		var helper *connhelper.ConnectionHelper

		helper, err = connhelper.GetConnectionHelper(dockerHost)
		if err != nil {
			return nil, err
		}
		cli, err = client.NewClientWithOpts(
			client.WithHost(helper.Host),
			client.WithDialContext(helper.Dialer),
			client.WithAPIVersionNegotiation(),
		)
	} else {
		cli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return cli, err
}

// isAttachedToNetwork return true if node is attached to network
func isAttachedToNetwork(node *k3d.Node, network string) bool {
	for _, nw := range node.Networks {
		if nw == network {
			return true
		}
	}
	return false
}
