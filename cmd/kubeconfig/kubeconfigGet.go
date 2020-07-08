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
package kubeconfig

import (
	"fmt"
	"os"

	"github.com/rancher/k3d/v3/cmd/util"
	"github.com/rancher/k3d/v3/pkg/cluster"
	"github.com/rancher/k3d/v3/pkg/runtimes"
	k3d "github.com/rancher/k3d/v3/pkg/types"
	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"
)

type getKubeconfigFlags struct {
	all bool
}

// NewCmdKubeconfigGet returns a new cobra command
func NewCmdKubeconfigGet() *cobra.Command {

	writeKubeConfigOptions := cluster.WriteKubeConfigOptions{
		UpdateExisting:       true,
		UpdateCurrentContext: true,
		OverwriteExisting:    true,
	}

	getKubeconfigFlags := getKubeconfigFlags{}

	// create new command
	cmd := &cobra.Command{
		Use:               "get [CLUSTER [CLUSTER [...]] | --all]",
		Short:             "Get kubeconfig from cluster(s).",
		Long:              `Get kubeconfig from cluster(s).`,
		ValidArgsFunction: util.ValidArgsAvailableClusters,
		Args: func(cmd *cobra.Command, args []string) error {
			if (len(args) < 1 && !getKubeconfigFlags.all) || (len(args) > 0 && getKubeconfigFlags.all) {
				return fmt.Errorf("Need to specify one or more cluster names *or* set `--all` flag")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			var clusters []*k3d.Cluster
			var err error

			// generate list of clusters
			if getKubeconfigFlags.all {
				clusters, err = cluster.GetClusters(cmd.Context(), runtimes.SelectedRuntime)
				if err != nil {
					log.Fatalln(err)
				}
			} else {
				for _, clusterName := range args {
					retrievedCluster, err := cluster.GetCluster(cmd.Context(), runtimes.SelectedRuntime, &k3d.Cluster{Name: clusterName})
					if err != nil {
						log.Fatalln(err)
					}
					clusters = append(clusters, retrievedCluster)
				}
			}

			// get kubeconfigs from all clusters
			errorGettingKubeconfig := false
			for _, c := range clusters {
				log.Debugf("Getting kubeconfig for cluster '%s'", c.Name)
				fmt.Println("---") // YAML document separator
				if _, err := cluster.GetAndWriteKubeConfig(cmd.Context(), runtimes.SelectedRuntime, c, "-", &writeKubeConfigOptions); err != nil {
					log.Errorln(err)
					errorGettingKubeconfig = true
				}
			}

			// return with non-zero exit code, if there was an error for one of the clusters
			if errorGettingKubeconfig {
				os.Exit(1)
			}
		},
	}

	// add flags
	cmd.Flags().BoolVarP(&getKubeconfigFlags.all, "all", "a", false, "Get kubeconfigs from all existing clusters")

	// done
	return cmd
}
