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
package tools

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	k3dc "github.com/rancher/k3d/v4/pkg/client"
	"github.com/rancher/k3d/v4/pkg/runtimes"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	"github.com/rancher/k3d/v4/version"
	log "github.com/sirupsen/logrus"
)

// ImageImportIntoClusterMulti starts up a k3d tools container for the selected cluster and uses it to export
// images from the runtime to import them into the nodes of the selected cluster
func ImageImportIntoClusterMulti(ctx context.Context, runtime runtimes.Runtime, images []string, cluster *k3d.Cluster, loadImageOpts k3d.ImageImportOpts) error {
	imagesFromRuntime, imagesFromTar, err := findImages(ctx, runtime, images)
	if err != nil {
		return err
	}

	// no images found to load -> exit early
	if len(imagesFromRuntime)+len(imagesFromTar) == 0 {
		return fmt.Errorf("No valid images specified")
	}

	cluster, err = k3dc.ClusterGet(ctx, runtime, cluster)
	if err != nil {
		log.Errorf("Failed to find the specified cluster")
		return err
	}

	if cluster.Network.Name == "" {
		return fmt.Errorf("Failed to get network for cluster '%s'", cluster.Name)
	}

	var imageVolume string
	var ok bool
	for _, node := range cluster.Nodes {
		if node.Role == k3d.ServerRole || node.Role == k3d.AgentRole {
			if imageVolume, ok = node.Labels[k3d.LabelImageVolume]; ok {
				break
			}
		}
	}
	if imageVolume == "" {
		return fmt.Errorf("Failed to find image volume for cluster '%s'", cluster.Name)
	}

	log.Debugf("Attaching to cluster's image volume '%s'", imageVolume)

	// create tools node to export images
	var toolsNode *k3d.Node
	toolsNode, err = runtime.GetNode(ctx, &k3d.Node{Name: fmt.Sprintf("%s-%s-tools", k3d.DefaultObjectNamePrefix, cluster.Name)})
	if err != nil || toolsNode == nil {
		log.Infoln("Starting k3d-tools node...")
		toolsNode, err = startToolsNode( // TODO: re-use existing container
			ctx,
			runtime,
			cluster,
			cluster.Network.Name,
			[]string{
				fmt.Sprintf("%s:%s", imageVolume, k3d.DefaultImageVolumeMountPath),
				fmt.Sprintf("%s:%s", runtime.GetRuntimePath(), runtime.GetRuntimePath()),
			})
		if err != nil {
			log.Errorf("Failed to start tools container for cluster '%s'", cluster.Name)
		}
	}

	/* TODO:
	 * Loop over list of images and check, whether they are files (tar archives) and sort them respectively
	 * Special case: '-' means "read from stdin"
	 * 1. From daemon: save images -> import
	 * 2. From file: copy file -> import
	 * 3. From stdin: save to tar -> import
	 * Note: temporary storage location is always the shared image volume and actions are always executed by the tools node
	 */
	var importTarNames []string

	if len(imagesFromRuntime) > 0 {
		// save image to tarfile in shared volume
		log.Infof("Saving %d image(s) from runtime...", len(imagesFromRuntime))
		tarName := fmt.Sprintf("%s/k3d-%s-images-%s.tar", k3d.DefaultImageVolumeMountPath, cluster.Name, time.Now().Format("20060102150405"))
		if err := runtime.ExecInNode(ctx, toolsNode, append([]string{"./k3d-tools", "save-image", "-d", tarName}, imagesFromRuntime...)); err != nil {
			log.Errorf("Failed to save image(s) in tools container for cluster '%s'", cluster.Name)
			return err
		}
		importTarNames = append(importTarNames, tarName)
	}

	if len(imagesFromTar) > 0 {
		// copy tarfiles to shared volume
		log.Infof("Saving %d tarball(s) to shared image volume...", len(imagesFromTar))
		for _, file := range imagesFromTar {
			tarName := fmt.Sprintf("%s/k3d-%s-images-%s-file-%s", k3d.DefaultImageVolumeMountPath, cluster.Name, time.Now().Format("20060102150405"), path.Base(file))
			if err := runtime.CopyToNode(ctx, file, tarName, toolsNode); err != nil {
				log.Errorf("Failed to copy image tar '%s' to tools node! Error below:\n%+v", file, err)
				continue
			}
			importTarNames = append(importTarNames, tarName)
		}
	}

	// import image in each node
	log.Infoln("Importing images into nodes...")
	var importWaitgroup sync.WaitGroup
	for _, tarName := range importTarNames {
		for _, node := range cluster.Nodes {
			// only import image in server and agent nodes (i.e. ignoring auxiliary nodes like the server loadbalancer)
			if node.Role == k3d.ServerRole || node.Role == k3d.AgentRole {
				importWaitgroup.Add(1)
				go func(node *k3d.Node, wg *sync.WaitGroup, tarPath string) {
					log.Infof("Importing images from tarball '%s' into node '%s'...", tarPath, node.Name)
					if err := runtime.ExecInNode(ctx, node, []string{"ctr", "image", "import", tarPath}); err != nil {
						log.Errorf("Failed to import images in node '%s'", node.Name)
						log.Errorln(err)
					}
					wg.Done()
				}(node, &importWaitgroup, tarName)
			}
		}
	}
	importWaitgroup.Wait()

	// remove tarball
	if !loadImageOpts.KeepTar && len(importTarNames) > 0 {
		log.Infoln("Removing the tarball(s) from image volume...")
		if err := runtime.ExecInNode(ctx, toolsNode, []string{"rm", "-f", strings.Join(importTarNames, " ")}); err != nil {
			log.Errorf("Failed to delete one or more tarballs from '%+v'", importTarNames)
			log.Errorln(err)
		}
	}

	// delete tools container
	log.Infoln("Removing k3d-tools node...")
	if err := runtime.DeleteNode(ctx, toolsNode); err != nil {
		log.Errorf("Failed to delete tools node '%s': Try to delete it manually", toolsNode.Name)
	}

	log.Infoln("Successfully imported image(s)")

	return nil

}

func findImages(ctx context.Context, runtime runtimes.Runtime, requestedImages []string) (imagesFromRuntime, imagesFromTar []string, err error) {
	runtimeImages, err := runtime.GetImages(ctx)
	if err != nil {
		log.Errorln("Failed to fetch list of existing images from runtime")
		return nil, nil, err
	}

	for _, requestedImage := range requestedImages {
		if isFile(requestedImage) {
			imagesFromTar = append(imagesFromTar, requestedImage)
			log.Debugf("Selected image '%s' is a file", requestedImage)
			break
		}

		runtimeImage, found := findRuntimeImage(requestedImage, runtimeImages)
		if found {
			imagesFromRuntime = append(imagesFromRuntime, runtimeImage)
			log.Debugf("Selected image '%s' (found as '%s') in runtime", requestedImage, runtimeImage)
			break
		}

		log.Warnf("Image '%s' is not a file and couldn't be found in the container runtime", requestedImage)
	}
	return imagesFromRuntime, imagesFromTar, err
}

func findRuntimeImage(requestedImage string, runtimeImages []string) (string, bool) {
	canonicalRequestedImage := canonicalImageName(requestedImage)

	for _, runtimeImage := range runtimeImages {
		if imageNamesEqual(canonicalRequestedImage, runtimeImage) {
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

// canonicalImageName turns any docker image name in the form `registry/orgName/imageName:versionName`.
// If `:versionName` is not present, `:latest` is suffixed.
// If `orgName` is not present, `library` is added.
// If `registry` is not present, `docker.io` is prefixed.
func canonicalImageName(image string) string {
	if !containsVersionPart(image) {
		image = fmt.Sprintf("%s:latest", image)
	}

	slashCount := strings.Count(image, "/")
	switch slashCount {
	case 0: // zero slashes -> library image tag (e.g. `postgres`)
		return fmt.Sprintf("docker.io/library/%s", image)
	case 1: // one slash -> no registry (e.g. `library/postgres`)
		return fmt.Sprintf("docker.io/%s", image)
	default: // two slashes or more -> already contains registry name
		return image
	}
}

func imageNamesEqual(requestedImageName string, runtimeImageName string) bool {
	if requestedImageName == runtimeImageName {
		return true
	}

	return requestedImageName == canonicalImageName(runtimeImageName)
}

func containsVersionPart(imageTag string) bool {
	// e.g. imageName has no colon -> false
	// e.g. repoName/imageName has no colon -> false
	// e.g. registry/repoName/imageName has no colon -> false
	// e.g. registry:1234/repoName/imageName has colon, but before the slash -> false
	// e.g. imageName:versionName has colon -> true
	// e.g. repoName/imageName:versionName has colon after slash -> true
	// e.g. registry/repoName/imageName:versionName has colon after slash -> true
	// e.g. registry:1234/repoName/imageName:versionName has colon after slash -> false
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

// startToolsNode will start a new k3d tools container and connect it to the network of the chosen cluster
func startToolsNode(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster, network string, volumes []string) (*k3d.Node, error) {
	labels := map[string]string{}
	for k, v := range k3d.DefaultObjectLabels {
		labels[k] = v
	}
	for k, v := range k3d.DefaultObjectLabelsVar {
		labels[k] = v
	}
	node := &k3d.Node{
		Name:     fmt.Sprintf("%s-%s-tools", k3d.DefaultObjectNamePrefix, cluster.Name),
		Image:    fmt.Sprintf("%s:%s", k3d.DefaultToolsImageRepo, version.GetHelperImageVersion()),
		Role:     k3d.NoRole,
		Volumes:  volumes,
		Networks: []string{network},
		Cmd:      []string{},
		Args:     []string{"noop"},
		Labels:   k3d.DefaultObjectLabels,
	}
	node.Labels[k3d.LabelClusterName] = cluster.Name
	if err := k3dc.NodeRun(ctx, runtime, node, k3d.NodeCreateOpts{}); err != nil {
		log.Errorf("Failed to create tools container for cluster '%s'", cluster.Name)
		return node, err
	}

	return node, nil
}
