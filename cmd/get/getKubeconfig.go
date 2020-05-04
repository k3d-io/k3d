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
package get

import (
	"fmt"
	"os"

	"github.com/rancher/k3d/pkg/cluster"
	"github.com/rancher/k3d/pkg/runtimes"
	k3d "github.com/rancher/k3d/pkg/types"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	log "github.com/sirupsen/logrus"
)

type getKubeconfigFlags struct {
	all    bool
	output string
}

// NewCmdGetKubeconfig returns a new cobra command
func NewCmdGetKubeconfig() *cobra.Command {

	writeKubeConfigOptions := cluster.WriteKubeConfigOptions{}

	getKubeconfigFlags := getKubeconfigFlags{}

	// create new command
	cmd := &cobra.Command{
		Use:   "kubeconfig [CLUSTER [CLUSTER [...]] | --all]", // TODO: getKubeconfig: allow more than one cluster name or even --all
		Short: "Get kubeconfig",
		Long:  `Get kubeconfig.`,
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
				clusters, err = cluster.GetClusters(runtimes.SelectedRuntime)
				if err != nil {
					log.Fatalln(err)
				}
			} else {
				for _, clusterName := range args {
					clusters = append(clusters, &k3d.Cluster{Name: clusterName})
				}
			}

			// get kubeconfigs from all clusters
			errorGettingKubeconfig := false
			for _, c := range clusters {
				log.Debugf("Getting kubeconfig for cluster '%s'", c.Name)
				if getKubeconfigFlags.output, err = cluster.GetAndWriteKubeConfig(runtimes.SelectedRuntime, c, getKubeconfigFlags.output, &writeKubeConfigOptions); err != nil {
					log.Errorln(err)
					errorGettingKubeconfig = true
				}
			}

			// only print kubeconfig file path if output is not stdout ("-")
			if getKubeconfigFlags.output != "-" {
				fmt.Println(getKubeconfigFlags.output)
			}

			// return with non-zero exit code, if there was an error for one of the clusters
			if errorGettingKubeconfig {
				os.Exit(1)
			}
		},
	}

	// add flags
	cmd.Flags().StringVarP(&getKubeconfigFlags.output, "output", "o", "", fmt.Sprintf("Define output [ - | FILE ] (default from $KUBECONFIG or %s", clientcmd.RecommendedHomeFile))
	if err := cmd.MarkFlagFilename("output"); err != nil {
		log.Fatalln("Failed to mark flag --output as filename")
	}
	cmd.Flags().BoolVarP(&writeKubeConfigOptions.UpdateExisting, "update", "u", true, "Update conflicting fields in existing KubeConfig")
	cmd.Flags().BoolVarP(&writeKubeConfigOptions.UpdateCurrentContext, "switch", "s", false, "Switch to new context")
	cmd.Flags().BoolVar(&writeKubeConfigOptions.OverwriteExisting, "overwrite", false, "[Careful!] Overwrite existing file, ignoring its contents")
	cmd.Flags().BoolVarP(&getKubeconfigFlags.all, "all", "a", false, "Get kubeconfigs from all existing clusters")

	// done
	return cmd
}
