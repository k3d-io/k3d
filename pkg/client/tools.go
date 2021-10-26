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
	"io"
	"os"
	"strings"
	"sync"

	l "github.com/rancher/k3d/v5/pkg/logger"
	"github.com/rancher/k3d/v5/pkg/runtimes"
	k3d "github.com/rancher/k3d/v5/pkg/types"
)

// ImageImportIntoClusterMulti starts up a k3d tools container for the selected cluster and uses it to export
// images from the runtime to import them into the nodes of the selected cluster
func ImageImportIntoClusterMulti(ctx context.Context, runtime runtimes.Runtime, images []string, cluster *k3d.Cluster, opts k3d.ImageImportOpts) error {

	// stdin case
	if len(images) == 1 && images[0] == "-" {
		loadImageFromStream(ctx, runtime, os.Stdin, cluster)
		return nil
	}

	imagesFromRuntime, imagesFromTar, err := findImages(ctx, runtime, images)
	if err != nil {
		return fmt.Errorf("failed to find images: %w", err)
	}

	// no images found to load -> exit early
	if len(imagesFromRuntime)+len(imagesFromTar) == 0 {
		return fmt.Errorf("No valid images specified")
	}

	/* TODO:
	 * Loop over list of images and check, whether they are files (tar archives) and sort them respectively
	 * Special case: '-' means "read from stdin"
	 * 1. From daemon: save images -> import
	 * 2. From file: copy file -> import
	 * 3. From stdin: save to tar -> import
	 * Note: temporary storage location is always the shared image volume and actions are always executed by the tools node
	 */

	if len(imagesFromRuntime) > 0 {
		l.Log().Infof("Loading %d image(s) from runtime into nodes...", len(imagesFromRuntime))
		// open a stream to all given images
		stream, err := runtime.GetImageStream(ctx, imagesFromRuntime)
		loadImageFromStream(ctx, runtime, stream, cluster)
		if err != nil {
			return fmt.Errorf("Could not open image stream for given images %s: %w", imagesFromRuntime, err)
		}
		// load the images directly into the nodes

	}

	// TODO: stdin for tar images as well
	if len(imagesFromTar) > 0 {
		// copy tarfiles to shared volume
		l.Log().Infof("Saving %d tarball(s) to shared image volume...", len(imagesFromTar))

		files := make([]*os.File, len(imagesFromTar))
		readers := make([]io.Reader, len(imagesFromTar))
		failedFiles := 0
		for i, fileName := range imagesFromTar {
			file, err := os.Open(fileName)
			if err != nil {
				l.Log().Errorf("failed to read file '%s', skipping. Error below:\n%+v", fileName, err)
				failedFiles++
				continue
			}
			files[i] = file
			readers[i] = file
		}
		multiReader := io.MultiReader(readers...)
		loadImageFromStream(ctx, runtime, io.NopCloser(multiReader), cluster)
		for _, file := range files {
			err := file.Close()
			if err != nil {
				l.Log().Errorf("Failed to close file '%s' after reading. Error below:\n%+v", file.Name(), err)
			}
		}
	}
	// TODO: stdin reading from args

	l.Log().Infoln("Successfully imported image(s)")
	return nil
}

func loadImageFromStream(ctx context.Context, runtime runtimes.Runtime, stream io.ReadCloser, cluster *k3d.Cluster) {
	var importWaitgroup sync.WaitGroup

	numNodes := 0
	for _, node := range cluster.Nodes {
		// only import image in server and agent nodes (i.e. ignoring auxiliary nodes like the server loadbalancer)
		if node.Role == k3d.ServerRole || node.Role == k3d.AgentRole {
			numNodes++
		}
	}
	// multiplex the stream so we can write to multiple nodes
	pipeReaders := make([]*io.PipeReader, numNodes)
	pipeWriters := make([]io.Writer, numNodes)
	for i := 0; i < numNodes; i++ {
		reader, writer := io.Pipe()
		pipeReaders[i] = reader
		pipeWriters[i] = writer
	}

	go func() {
		_, err := io.Copy(io.MultiWriter(pipeWriters...), stream)
		if err != nil {
			l.Log().Errorf("Failed to copy read stream. %v", err)
		}
	}()

	pipeId := 0
	for _, node := range cluster.Nodes {
		// only import image in server and agent nodes (i.e. ignoring auxiliary nodes like the server loadbalancer)
		if node.Role == k3d.ServerRole || node.Role == k3d.AgentRole {
			importWaitgroup.Add(1)
			go func(node *k3d.Node, wg *sync.WaitGroup, stream io.ReadCloser) {
				l.Log().Infof("Importing images into node '%s'...", node.Name)
				if err := runtime.ExecInNodeWithStdin(ctx, node, []string{"ctr", "image", "import", "-"}, stream); err != nil {
					l.Log().Errorf("failed to import images in node '%s': %v", node.Name, err)
				}
				wg.Done()
			}(node, &importWaitgroup, pipeReaders[pipeId])
			pipeId++
		}
	}
	importWaitgroup.Wait()
}

type runtimeImageGetter interface {
	GetImages(context.Context) ([]string, error)
}

func findImages(ctx context.Context, runtime runtimeImageGetter, requestedImages []string) (imagesFromRuntime, imagesFromTar []string, err error) {
	runtimeImages, err := runtime.GetImages(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch list of existing images from runtime: %w", err)
	}

	for _, requestedImage := range requestedImages {
		if isFile(requestedImage) {
			imagesFromTar = append(imagesFromTar, requestedImage)
			l.Log().Debugf("Selected image '%s' is a file", requestedImage)
			continue
		}

		runtimeImage, found := findRuntimeImage(requestedImage, runtimeImages)
		if found {
			imagesFromRuntime = append(imagesFromRuntime, runtimeImage)
			l.Log().Debugf("Selected image '%s' (found as '%s') in runtime", requestedImage, runtimeImage)
			continue
		}

		l.Log().Warnf("Image '%s' is not a file and couldn't be found in the container runtime", requestedImage)
	}
	return imagesFromRuntime, imagesFromTar, nil
}

func findRuntimeImage(requestedImage string, runtimeImages []string) (string, bool) {
	for _, runtimeImage := range runtimeImages {
		if imageNamesEqual(requestedImage, runtimeImage) {
			return runtimeImage, true
		}
	}

	// if not found, check for special Docker image naming
	for _, runtimeImage := range runtimeImages {
		if dockerSpecialImageNameEqual(requestedImage, runtimeImage) {
			return runtimeImage, true
		}
	}
	return "", false
}

func isFile(image string) bool {
	file, err := os.Stat(image)
	if err != nil {
		return false
	}
	return !file.IsDir()
}

func dockerSpecialImageNameEqual(requestedImageName string, runtimeImageName string) bool {
	if strings.HasPrefix(requestedImageName, "docker.io/") {
		return dockerSpecialImageNameEqual(strings.TrimPrefix(requestedImageName, "docker.io/"), runtimeImageName)
	}

	if strings.HasPrefix(requestedImageName, "library/") {
		return imageNamesEqual(strings.TrimPrefix(requestedImageName, "library/"), runtimeImageName)
	}

	return false
}

func imageNamesEqual(requestedImageName string, runtimeImageName string) bool {
	// first, compare what the user provided
	if requestedImageName == runtimeImageName {
		return true
	}

	// transform to canonical image name, i.e. ensure `:versionName` part on both ends
	return canonicalImageName(requestedImageName) == runtimeImageName
}

// canonicalImageName adds `:latest` suffix if `:anyOtherVersionName` is not present.
func canonicalImageName(image string) string {
	if !containsVersionPart(image) {
		image = fmt.Sprintf("%s:latest", image)
	}
	return image
}

func containsVersionPart(imageTag string) bool {
	if !strings.Contains(imageTag, ":") {
		return false
	}

	if !strings.Contains(imageTag, "/") {
		// happens if someone refers to a library image by just it's imageName (e.g. `postgres` instead of `library/postgres`)
		return strings.Contains(imageTag, ":")
	}

	indexOfSlash := strings.Index(imageTag, "/") // can't be -1 because the existence of a '/' is ensured above
	substringAfterSlash := imageTag[indexOfSlash:]
	return strings.Contains(substringAfterSlash, ":")
}

// runToolsNode will start a new k3d tools container and connect it to the network of the chosen cluster
func runToolsNode(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster, network string, volumes []string) (*k3d.Node, error) {
	labels := map[string]string{}
	for k, v := range k3d.DefaultRuntimeLabels {
		labels[k] = v
	}
	for k, v := range k3d.DefaultRuntimeLabelsVar {
		labels[k] = v
	}
	node := &k3d.Node{
		Name:          fmt.Sprintf("%s-%s-tools", k3d.DefaultObjectNamePrefix, cluster.Name),
		Image:         k3d.GetToolsImage(),
		Role:          k3d.NoRole,
		Volumes:       volumes,
		Networks:      []string{network},
		Cmd:           []string{},
		Args:          []string{"noop"},
		RuntimeLabels: labels,
	}
	node.RuntimeLabels[k3d.LabelClusterName] = cluster.Name
	if err := NodeRun(ctx, runtime, node, k3d.NodeCreateOpts{}); err != nil {
		return node, fmt.Errorf("failed to run k3d-tools node for cluster '%s': %w", cluster.Name, err)
	}

	return node, nil
}

func EnsureToolsNode(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster) (*k3d.Node, error) {

	var toolsNode *k3d.Node
	toolsNode, err := runtime.GetNode(ctx, &k3d.Node{Name: fmt.Sprintf("%s-%s-tools", k3d.DefaultObjectNamePrefix, cluster.Name)})
	if err != nil || toolsNode == nil {

		// Get more info on the cluster, if required
		var imageVolume string
		if cluster.Network.Name == "" || cluster.ImageVolume == "" {
			l.Log().Debugf("Gathering some more info about the cluster before creating the tools node...")
			var err error
			cluster, err = ClusterGet(ctx, runtime, cluster)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve cluster: %w", err)
			}

			if cluster.Network.Name == "" {
				return nil, fmt.Errorf("failed to get network for cluster '%s'", cluster.Name)
			}

			var ok bool
			for _, node := range cluster.Nodes {
				if node.Role == k3d.ServerRole || node.Role == k3d.AgentRole {
					if imageVolume, ok = node.RuntimeLabels[k3d.LabelImageVolume]; ok {
						break
					}
				}
			}
			if imageVolume == "" {
				return nil, fmt.Errorf("Failed to find image volume for cluster '%s'", cluster.Name)
			}
			l.Log().Debugf("Attaching to cluster's image volume '%s'", imageVolume)
			cluster.ImageVolume = imageVolume
		}

		// start tools node
		l.Log().Infoln("Starting new tools node...")
		toolsNode, err = runToolsNode(
			ctx,
			runtime,
			cluster,
			cluster.Network.Name,
			[]string{
				fmt.Sprintf("%s:%s", cluster.ImageVolume, k3d.DefaultImageVolumeMountPath),
				fmt.Sprintf("%s:%s", runtime.GetRuntimePath(), runtime.GetRuntimePath()),
			})
		if err != nil {
			l.Log().Errorf("Failed to run tools container for cluster '%s'", cluster.Name)
		}
	} else if !toolsNode.State.Running {
		l.Log().Infof("Starting existing tools node %s...", toolsNode.Name)
		if err := runtime.StartNode(ctx, toolsNode); err != nil {
			return nil, fmt.Errorf("error starting existing tools node %s: %v", toolsNode.Name, err)
		}
	}

	return toolsNode, err

}
