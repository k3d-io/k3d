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
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/docker/docker/api/types/container"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	runtimeErrors "github.com/k3d-io/k3d/v5/pkg/runtimes/errors"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

func IsDockerDesktop(os string) bool {
	return strings.ToLower(os) == "docker desktop"
}

/*
 * Simple Matching to detect local connection:
 * - file (socket): starts with / (absolute path)
 * - tcp://(localhost|127.0.0.1)
 * - ssh://(localhost|127.0.0.1)
 */
var LocalConnectionRegexp = regexp.MustCompile(`^(/|((tcp|ssh)://(localhost|127\.0\.0\.1))).*`)

func IsLocalConnection(endpoint string) bool {
	return LocalConnectionRegexp.Match([]byte(endpoint))
}

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
		return fmt.Errorf("failed to get docker client: %w", err)
	}
	defer docker.Close()

	nodeContainer, err := getNodeContainer(ctx, node)
	if err != nil {
		return fmt.Errorf("failed to find container for target node '%s': %w", node.Name, err)
	}

	// source: docker/cli/cli/command/container/cp
	srcInfo, err := archive.CopyInfoSourcePath(src, false)
	if err != nil {
		return fmt.Errorf("failed to copy info source path: %w", err)
	}

	srcArchive, err := archive.TarResource(srcInfo)
	if err != nil {
		return fmt.Errorf("failed to create tar resource: %w", err)
	}
	defer srcArchive.Close()

	destInfo := archive.CopyInfo{Path: dest}

	destStat, _ := docker.ContainerStatPath(ctx, nodeContainer.ID, dest) // don't blame me, docker is also not doing anything if err != nil ¯\_(ツ)_/¯

	destInfo.Exists, destInfo.IsDir = true, destStat.Mode.IsDir()

	destDir, preparedArchive, err := archive.PrepareArchiveCopy(srcArchive, srcInfo, destInfo)
	if err != nil {
		return fmt.Errorf("failed to prepare archive: %w", err)
	}
	defer preparedArchive.Close()

	return docker.CopyToContainer(ctx, nodeContainer.ID, destDir, preparedArchive, container.CopyToContainerOptions{AllowOverwriteDirWithFile: false})
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
		return fmt.Errorf("failed to get docker client: %w", err)
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
		l.Log().Debugf("Failed to close tar writer: %+v", err)
	}

	tarBytes := bytes.NewReader(buf.Bytes())
	if err := docker.CopyToContainer(ctx, nodeContainer.ID, "/", tarBytes, container.CopyToContainerOptions{AllowOverwriteDirWithFile: true}); err != nil {
		return fmt.Errorf("Failed to copy content to container '%s': %+v", nodeContainer.ID, err)
	}

	return nil
}

// ReadFromNode reads from a given filepath inside the node container
func (d Docker) ReadFromNode(ctx context.Context, path string, node *k3d.Node) (io.ReadCloser, error) {
	l.Log().Tracef("Reading path %s from node %s...", path, node.Name)
	nodeContainer, err := getNodeContainer(ctx, node)
	if err != nil {
		return nil, fmt.Errorf("failed to find container for node '%s': %w", node.Name, err)
	}

	docker, err := GetDockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get docker client: %w", err)
	}
	defer docker.Close()

	reader, _, err := docker.CopyFromContainer(ctx, nodeContainer.ID, path)
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil, errors.Wrap(runtimeErrors.ErrRuntimeFileNotFound, err.Error())
		}
		return nil, fmt.Errorf("failed to copy path '%s' from container '%s': %w", path, nodeContainer.ID, err)
	}

	return reader, err
}

// GetDockerClient returns a docker client
func GetDockerClient() (client.APIClient, error) {
	dockerCli, err := command.NewDockerCli(command.WithStandardStreams())
	if err != nil {
		return nil, fmt.Errorf("failed to create new docker CLI with standard streams: %w", err)
	}

	newClientOpts := flags.NewClientOptions()
	newClientOpts.LogLevel = l.Log().GetLevel().String() // this is needed, as the following Initialize() call will set a new log level on the global logrus instance

	flagset := pflag.NewFlagSet("docker", pflag.ContinueOnError)
	newClientOpts.InstallFlags(flagset)
	newClientOpts.SetDefaultOptions(flagset)

	err = dockerCli.Initialize(newClientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize docker CLI: %w", err)
	}

	return dockerCli.Client(), nil
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
