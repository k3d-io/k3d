/*
Copyright © 2020-2024 The k3d Author(s)

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
	"os"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3drt "github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/pkg/types/k3s"
	"github.com/k3d-io/k3d/v5/pkg/util"
)

func ClusterPrepManifestVolume(ctx context.Context, runtime k3drt.Runtime, cluster *k3d.Cluster, clusterCreateOpts *k3d.ClusterCreateOpts) error {
	/*
	 * Server Manifests volume
	 * - manifest volume (for auto-deploy manifests)
	 */
	manifestVolumeName := fmt.Sprintf("%s-%s-manifests", k3d.DefaultObjectNamePrefix, cluster.Name)
	if err := runtime.CreateVolume(ctx, manifestVolumeName, map[string]string{k3d.LabelClusterName: cluster.Name}); err != nil {
		return fmt.Errorf("failed to create manifest volume '%s' for cluster '%s': %w", manifestVolumeName, cluster.Name, err)
	}
	l.Log().Infof("Created manifest volume %s", manifestVolumeName)

	clusterCreateOpts.GlobalLabels[k3d.LabelManifestVolume] = manifestVolumeName
	cluster.ManifestVolume = manifestVolumeName
	cluster.Volumes = append(cluster.Volumes, manifestVolumeName)

	// Attach volume to server nodes only
	filteredNodes, err := util.FilterNodes(cluster.Nodes, []string{"server:*"})
	if err != nil {
		return fmt.Errorf("failed to filter nodes: %w", err)
	}
	for _, node := range filteredNodes {
		node.Volumes = append(node.Volumes, fmt.Sprintf("%s:%s", manifestVolumeName, k3s.K3sPathManifestsEmbedded))
	}
	l.Log().Debugf("Attached manifest volume to server nodes")

	return nil
}

func CopyManifetsToVolume(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster, toolsNode *k3d.Node) error {
	for _, manifest := range cluster.Manifests {
		onContainerManifestFileName := k3s.K3sPathManifestsEmbedded + "/" + manifest.Name
		onHostManifestFile, err := os.CreateTemp("", manifest.Name)
		if err != nil {
			return fmt.Errorf("failed to create manifest temporary file: %w", err)
		}
		l.Log().Debugf("Created temporary local manifest file: %s", onHostManifestFile.Name())

		if _, err = onHostManifestFile.WriteString(manifest.Manifest); err != nil {
			return fmt.Errorf("Failed to write to output file: %+v", err)
		}

		if err := runtime.CopyToNode(ctx, onHostManifestFile.Name(), onContainerManifestFileName, toolsNode); err != nil {
			return fmt.Errorf("failed to copy manifestFile tar '%s' to tools node! Error below:\n%+v",
				onHostManifestFile.Name(), err)
		}

		os.Remove(onHostManifestFile.Name())
		l.Log().Debugf("Removed temporary local manifest file: %s", onHostManifestFile.Name())
	}

	return nil
}
