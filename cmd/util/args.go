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
package util

import (
	"context"

	k3dcluster "github.com/rancher/k3d/pkg/cluster"
	"github.com/rancher/k3d/pkg/runtimes"

	log "github.com/sirupsen/logrus"

	k3d "github.com/rancher/k3d/pkg/types"
)

// BuildClusterList return Cluster build from argument or all cluster if no args are supply
func BuildClusterList(ctx context.Context, args []string) []*k3d.Cluster {
	var clusters []*k3d.Cluster
	var err error

	if len(args) == 0 {
		// cluster name not specified : get all clusters
		clusters, err = k3dcluster.GetClusters(ctx, runtimes.SelectedRuntime)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		for _, clusterName := range args {
			// cluster name specified : get specific cluster
			retrievedCluster, err := k3dcluster.GetCluster(ctx, runtimes.SelectedRuntime, &k3d.Cluster{Name: clusterName})
			if err != nil {
				// if failed to retrieve one cluster print error in stderr but continue with other cluster
				log.Errorln(err)
			} else {
				clusters = append(clusters, retrievedCluster)
			}
		}
	}

	return clusters
}
