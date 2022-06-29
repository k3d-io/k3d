/*
Copyright © 2020-2022 The k3d Author(s)

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
	"path"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

// ImageImportIntoClusterMulti starts up a k3d tools container for the selected cluster and uses it to export
// images from the runtime to import them into the nodes of the selected cluster
func ImageImportIntoClusterMulti(ctx context.Context, runtime runtimes.Runtime, images []string, cluster *k3d.Cluster, opts k3d.ImageImportOpts) error {

	// stdin case
	if len(images) == 1 && images[0] == "-" {
		err := loadImageFromStream(ctx, runtime, os.Stdin, cluster, []string{"stdin"})
		return fmt.Errorf("failed to load image to cluster from stdin: %v", err)
	}

	imagesFromRuntime, imagesFromTar, err := findImages(ctx, runtime, images)
	if err != nil {
		return fmt.Errorf("failed to find images: %w", err)
	}

	// no images found to load -> exit early
	if len(imagesFromRuntime)+len(imagesFromTar) == 0 {
		return fmt.Errorf("no valid images specified")
	}

	loadWithToolsNode := false

	switch opts.Mode {
	case k3d.ImportModeAutoDetect:
		if err != nil {
			return fmt.Errorf("failed to retrieve container runtime information: %w", err)
		}
		runtimeHost := runtime.GetHost()
		if runtimeHost != "" && runtimeHost != "localhost" && runtimeHost != "127.0.0.1" {
			l.Log().Infof("Auto-detected a remote docker daemon, using tools node for loading images")
			loadWithToolsNode = true
		}
	case k3d.ImportModeToolsNode:
		loadWithToolsNode = true
	case k3d.ImportModeDirect:
		loadWithToolsNode = false
	}

	/* TODO:
	 * Loop over list of images and check, whether they are files (tar archives) and sort them respectively
	 * Special case: '-' means "read from stdin"
	 * 1. From daemon: save images -> import
	 * 2. From file: copy file -> import
	 * 3. From stdin: save to tar -> import
	 * Note: temporary storage location is always the shared image volume and actions are always executed by the tools node
	 */
	if loadWithToolsNode {
		err = importWithToolsNode(ctx, runtime, cluster, imagesFromRuntime, imagesFromTar, opts)
	} else {
		err = importWithStream(ctx, runtime, cluster, imagesFromRuntime, imagesFromTar)
	}
	if err != nil {
		return err
	}

	l.Log().Infoln("Successfully imported image(s)")
	return nil
}

func importWithToolsNode(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster, imagesFromRuntime []string, imagesFromTar []string, opts k3d.ImageImportOpts) error {
	// create tools node to export images
	toolsNode, err := EnsureToolsNode(ctx, runtime, cluster)
	if err != nil {
		return fmt.Errorf("failed to ensure that tools node is running: %w", err)
	}

	var importTarNames []string

	if len(imagesFromRuntime) > 0 {
		// save image to tarfile in shared volume
		l.Log().Infof("Saving %d image(s) from runtime...", len(imagesFromRuntime))
		tarName := fmt.Sprintf("%s/k3d-%s-images-%s.tar", k3d.DefaultImageVolumeMountPath, cluster.Name, time.Now().Format("20060102150405"))
		if err := runtime.ExecInNode(ctx, toolsNode, append([]string{"./k3d-tools", "save-image", "-d", tarName}, imagesFromRuntime...)); err != nil {
			return fmt.Errorf("failed to save image(s) in tools container for cluster '%s': %w", cluster.Name, err)
		}
		importTarNames = append(importTarNames, tarName)
	}

	if len(imagesFromTar) > 0 {
		// copy tarfiles to shared volume
		l.Log().Infof("Saving %d tarball(s) to shared image volume...", len(imagesFromTar))
		for _, file := range imagesFromTar {
			tarName := fmt.Sprintf("%s/k3d-%s-images-%s-file-%s", k3d.DefaultImageVolumeMountPath, cluster.Name, time.Now().Format("20060102150405"), path.Base(file))
			if err := runtime.CopyToNode(ctx, file, tarName, toolsNode); err != nil {
				l.Log().Errorf("failed to copy image tar '%s' to tools node! Error below:\n%+v", file, err)
				continue
			}
			importTarNames = append(importTarNames, tarName)
		}
	}

	// import image in each node
	l.Log().Infoln("Importing images into nodes...")
	var importWaitgroup sync.WaitGroup
	for _, tarName := range importTarNames {
		for _, node := range cluster.Nodes {
			// only import image in server and agent nodes (i.e. ignoring auxiliary nodes like the server loadbalancer)
			if node.Role == k3d.ServerRole || node.Role == k3d.AgentRole {
				importWaitgroup.Add(1)
				go func(node *k3d.Node, wg *sync.WaitGroup, tarPath string) {
					l.Log().Infof("Importing images from tarball '%s' into node '%s'...", tarPath, node.Name)
					if err := runtime.ExecInNode(ctx, node, []string{"ctr", "image", "import", tarPath}); err != nil {
						l.Log().Errorf("failed to import images in node '%s': %v", node.Name, err)
					}
					wg.Done()
				}(node, &importWaitgroup, tarName)
			}
		}
	}
	importWaitgroup.Wait()

	// remove tarball
	if !opts.KeepTar && len(importTarNames) > 0 {
		l.Log().Infoln("Removing the tarball(s) from image volume...")
		if err := runtime.ExecInNode(ctx, toolsNode, []string{"rm", "-f", strings.Join(importTarNames, " ")}); err != nil {
			l.Log().Errorf("failed to delete one or more tarballs from '%+v': %v", importTarNames, err)
		}
	}

	// delete tools container
	if !opts.KeepToolsNode {
		l.Log().Infoln("Removing k3d-tools node...")
		if err := runtime.DeleteNode(ctx, toolsNode); err != nil {
			l.Log().Errorf("failed to delete tools node '%s' (try to delete it manually): %v", toolsNode.Name, err)
		}
	}
	return nil
}

func importWithStream(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster, imagesFromRuntime []string, imagesFromTar []string) error {
	if len(imagesFromRuntime) > 0 {
		l.Log().Infof("Loading %d image(s) from runtime into nodes...", len(imagesFromRuntime))
		// open a stream to all given images
		stream, err := runtime.GetImageStream(ctx, imagesFromRuntime)
		if err != nil {
			return fmt.Errorf("could not open image stream for given images %s: %w", imagesFromRuntime, err)
		}
		err = loadImageFromStream(ctx, runtime, stream, cluster, imagesFromRuntime)
		if err != nil {
			return fmt.Errorf("could not load image to cluster from stream %s: %w", imagesFromRuntime, err)
		}
		// load the images directly into the nodes

	}

	if len(imagesFromTar) > 0 {
		// copy tarfiles to shared volume
		l.Log().Infof("Importing images from %d tarball(s)...", len(imagesFromTar))

		for _, fileName := range imagesFromTar {
			file, err := os.Open(fileName)
			if err != nil {
				return err
			}
			err = loadImageFromStream(ctx, runtime, file, cluster, []string{fileName})
			if err != nil {
				return fmt.Errorf("could not load image to cluster from stream %s: %w", fileName, err)
			}
		}
	}
	return nil
}

func loadImageFromStream(ctx context.Context, runtime runtimes.Runtime, stream io.ReadCloser, cluster *k3d.Cluster, imageNames []string) error {
	var errorGroup errgroup.Group

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

	errorGroup.Go(func() error {
		_, err := io.Copy(io.MultiWriter(pipeWriters...), stream)
		if err != nil {
			return fmt.Errorf("failed to copy read stream. %v", err)
		}
		err = stream.Close()
		if err != nil {
			return fmt.Errorf("failed to close stream. %v", err)
		}
		return nil
	})

	pipeId := 0
	for _, n := range cluster.Nodes {
		node := n
		// only import image in server and agent nodes (i.e. ignoring auxiliary nodes like the server loadbalancer)
		if node.Role == k3d.ServerRole || node.Role == k3d.AgentRole {
			pipeReader := pipeReaders[pipeId]
			errorGroup.Go(func() error {
				l.Log().Infof("Importing images '%s' into node '%s'...", imageNames, node.Name)
				if err := runtime.ExecInNodeWithStdin(ctx, node, []string{"ctr", "image", "import", "-"}, pipeReader); err != nil {
					return fmt.Errorf("failed to import images in node '%s': %v", node.Name, err)
				}
				return nil
			})
			pipeId++
		}
	}
	err := errorGroup.Wait()
	if err != nil {
		return fmt.Errorf("error loading image to cluster, first error: %v", err)
	}
	return nil
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
	prefixes := []string{
		"docker.io/library/",
		"docker.io/",
		"library/",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(requestedImageName, prefix) {
			return imageNamesEqual(strings.TrimPrefix(requestedImageName, prefix), runtimeImageName)
		}

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
		ExtraHosts:    []string{fmt.Sprintf("%s:host-gateway", k3d.DefaultK3dInternalHostRecord)},
	}
	node.RuntimeLabels[k3d.LabelClusterName] = cluster.Name
	if err := NodeRun(ctx, runtime, node, k3d.NodeCreateOpts{}); err != nil {
		return node, fmt.Errorf("failed to run k3d-tools node for cluster '%s': %w", cluster.Name, err)
	}

	return node, nil
}

var EnsureToolsNodeMutex sync.Mutex

func EnsureToolsNode(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster) (*k3d.Node, error) {
	// Mutex to prevent simultaneous creation of two tools-nodes, e.g. due to slow network connection cause the image pulling to take too long
	// FIXME: could be prevented completely by having a smarter image pre-pulling mechanism
	EnsureToolsNodeMutex.Lock()
	defer EnsureToolsNodeMutex.Unlock()

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
