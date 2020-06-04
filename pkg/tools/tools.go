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
package tools

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	k3dc "github.com/rancher/k3d/pkg/cluster"
	"github.com/rancher/k3d/pkg/runtimes"
	k3d "github.com/rancher/k3d/pkg/types"
	log "github.com/sirupsen/logrus"
)

// LoadImagesIntoCluster starts up a k3d tools container for the selected cluster and uses it to export
// images from the runtime to import them into the nodes of the selected cluster
func LoadImagesIntoCluster(ctx context.Context, runtime runtimes.Runtime, images []string, cluster *k3d.Cluster, loadImageOpts k3d.LoadImageOpts) error {

	var imagesFromRuntime []string
	var imagesFromTar []string

	runtimeImages, err := runtime.GetImages(ctx)
	if err != nil {
		log.Errorln("Failed to fetch list of exsiting images from runtime")
		return err
	}

	for _, image := range images {
		found := false
		// Check if the current element is a file
		if _, err := os.Stat(image); os.IsNotExist(err) {
			// not a file? Check if such an image is present in the container runtime
			for _, runtimeImage := range runtimeImages {
				if image == runtimeImage {
					found = true
					imagesFromRuntime = append(imagesFromRuntime, image)
					log.Debugf("Selected image '%s' found in runtime", image)
					break
				}
			}
		} else {
			// file exists
			found = true
			imagesFromTar = append(imagesFromTar, image)
			log.Debugf("Selected image '%s' is a file", image)
		}
		if !found {
			log.Warnf("Image '%s' is not a file and couldn't be found in the container runtime", image)
		}
	}

	cluster, err = k3dc.GetCluster(ctx, runtime, cluster)
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
		if node.Role == k3d.MasterRole || node.Role == k3d.WorkerRole {
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
			// only import image in master and worker nodes (i.e. ignoring auxiliary nodes like the master loadbalancer)
			if node.Role == k3d.MasterRole || node.Role == k3d.WorkerRole {
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
		log.Errorln("Failed to delete tools node '%s': Try to delete it manually", toolsNode.Name)
	}

	log.Infoln("Successfully imported image(s)")

	return nil

}

// startToolsNode will start a new k3d tools container and connect it to the network of the chosen cluster
func startToolsNode(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster, network string, volumes []string) (*k3d.Node, error) {
	node := &k3d.Node{
		Name:    fmt.Sprintf("%s-%s-tools", k3d.DefaultObjectNamePrefix, cluster.Name),
		Image:   k3d.DefaultToolsContainerImage,
		Role:    k3d.NoRole,
		Volumes: volumes,
		Network: network,
		Cmd:     []string{},
		Args:    []string{"noop"},
		Labels:  k3d.DefaultObjectLabels,
	}
	node.Labels["k3d.cluster"] = cluster.Name
	if err := runtime.CreateNode(ctx, node); err != nil {
		log.Errorf("Failed to create tools container for cluster '%s'", cluster.Name)
		return node, err
	}

	return node, nil
}
