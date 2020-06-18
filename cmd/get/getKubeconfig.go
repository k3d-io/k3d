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
	"path"
	"strings"

	"github.com/rancher/k3d/v3/cmd/util"
	"github.com/rancher/k3d/v3/pkg/cluster"
	"github.com/rancher/k3d/v3/pkg/runtimes"
	k3d "github.com/rancher/k3d/v3/pkg/types"
	k3dutil "github.com/rancher/k3d/v3/pkg/util"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	log "github.com/sirupsen/logrus"
)

type getKubeconfigFlags struct {
	all           bool
	output        string
	targetDefault bool
}

// NewCmdGetKubeconfig returns a new cobra command
func NewCmdGetKubeconfig() *cobra.Command {

	writeKubeConfigOptions := cluster.WriteKubeConfigOptions{}

	getKubeconfigFlags := getKubeconfigFlags{}

	// create new command
	cmd := &cobra.Command{
		Use:               "kubeconfig [CLUSTER [CLUSTER [...]] | --all]", // TODO: getKubeconfig: allow more than one cluster name or even --all
		Short:             "Get kubeconfig",
		Long:              `Get kubeconfig.`,
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

			if getKubeconfigFlags.targetDefault && getKubeconfigFlags.output != "" {
				log.Fatalln("Cannot use both '--output' and '--default' at the same time")
			}

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
			var outputs []string
			outputDir, err := k3dutil.GetConfigDirOrCreate()
			if err != nil {
				log.Errorln(err)
				log.Fatalln("Failed to save kubeconfig to local directory")
			}
			for _, c := range clusters {
				log.Debugf("Getting kubeconfig for cluster '%s'", c.Name)
				output := getKubeconfigFlags.output
				if output == "" && !getKubeconfigFlags.targetDefault {
					output = path.Join(outputDir, fmt.Sprintf("kubeconfig-%s.yaml", c.Name))
				}
				output, err = cluster.GetAndWriteKubeConfig(cmd.Context(), runtimes.SelectedRuntime, c, output, &writeKubeConfigOptions)
				if err != nil {
					log.Errorln(err)
					errorGettingKubeconfig = true
				} else {
					outputs = append(outputs, output)
				}
			}

			// only print kubeconfig file path if output is not stdout ("-")
			if getKubeconfigFlags.output != "-" {
				fmt.Println(strings.Join(outputs, ":"))
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
	cmd.Flags().BoolVarP(&getKubeconfigFlags.targetDefault, "default-kubeconfig", "d", false, fmt.Sprintf("Update the default kubeconfig ($KUBECONFIG or %s)", clientcmd.RecommendedHomeFile))
	cmd.Flags().BoolVarP(&writeKubeConfigOptions.UpdateExisting, "update", "u", true, "Update conflicting fields in existing KubeConfig")
	cmd.Flags().BoolVarP(&writeKubeConfigOptions.UpdateCurrentContext, "switch", "s", true, "Switch to new context")
	cmd.Flags().BoolVar(&writeKubeConfigOptions.OverwriteExisting, "overwrite", false, "[Careful!] Overwrite existing file, ignoring its contents")
	cmd.Flags().BoolVarP(&getKubeconfigFlags.all, "all", "a", false, "Get kubeconfigs from all existing clusters")

	// done
	return cmd
}
