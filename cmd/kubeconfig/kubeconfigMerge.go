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
package kubeconfig

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/k3d-io/k3d/v5/cmd/util"
	"github.com/k3d-io/k3d/v5/pkg/client"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	k3dutil "github.com/k3d-io/k3d/v5/pkg/util"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

type mergeKubeconfigFlags struct {
	all           bool
	output        string
	targetDefault bool
}

// NewCmdKubeconfigMerge returns a new cobra command
func NewCmdKubeconfigMerge() *cobra.Command {
	writeKubeConfigOptions := client.WriteKubeConfigOptions{}

	mergeKubeconfigFlags := mergeKubeconfigFlags{}

	// create new command
	cmd := &cobra.Command{
		Use:               "merge [CLUSTER [CLUSTER [...]] | --all]",
		Aliases:           []string{"write"},
		Long:              `Write/Merge kubeconfig(s) from cluster(s) into new or existing kubeconfig/file.`,
		Short:             "Write/Merge kubeconfig(s) from cluster(s) into new or existing kubeconfig/file.",
		ValidArgsFunction: util.ValidArgsAvailableClusters,
		Args:              cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			var clusters []*k3d.Cluster
			var err error

			if mergeKubeconfigFlags.targetDefault && mergeKubeconfigFlags.output != "" {
				l.Log().Fatalln("Cannot use both '--output' and '--kubeconfig-merge-default' at the same time")
			}

			// generate list of clusters
			if mergeKubeconfigFlags.all {
				clusters, err = client.ClusterList(cmd.Context(), runtimes.SelectedRuntime)
				if err != nil {
					l.Log().Fatalln(err)
				}
			} else {
				clusternames := []string{k3d.DefaultClusterName}
				if len(args) != 0 {
					clusternames = args
				}

				for _, clusterName := range clusternames {
					retrievedCluster, err := client.ClusterGet(cmd.Context(), runtimes.SelectedRuntime, &k3d.Cluster{Name: clusterName})
					if err != nil {
						l.Log().Fatalln(err)
					}
					clusters = append(clusters, retrievedCluster)
				}
			}

			// get kubeconfigs from all clusters
			errorGettingKubeconfig := false
			var outputs []string
			outputDir, err := k3dutil.GetConfigDirOrCreate()
			if err != nil {
				l.Log().Errorln(err)
				l.Log().Fatalln("Failed to save kubeconfig to local directory")
			}
			for _, c := range clusters {
				l.Log().Debugf("Getting kubeconfig for cluster '%s'", c.Name)
				output := mergeKubeconfigFlags.output
				if output == "" && !mergeKubeconfigFlags.targetDefault {
					output = path.Join(outputDir, fmt.Sprintf("kubeconfig-%s.yaml", c.Name))
				}
				output, err = client.KubeconfigGetWrite(cmd.Context(), runtimes.SelectedRuntime, c, output, &writeKubeConfigOptions)
				if err != nil {
					l.Log().Errorln(err)
					errorGettingKubeconfig = true
				} else {
					outputs = append(outputs, output)
				}
			}

			// only print kubeconfig file path if output is not stdout ("-")
			if mergeKubeconfigFlags.output != "-" {
				fmt.Println(strings.Join(outputs, ":"))
			}

			// return with non-zero exit code, if there was an error for one of the clusters
			if errorGettingKubeconfig {
				os.Exit(1)
			}
		},
	}

	// add flags
	cmd.Flags().StringVarP(&mergeKubeconfigFlags.output, "output", "o", "", fmt.Sprintf("Define output [ - | FILE ] (default from $KUBECONFIG or %s", clientcmd.RecommendedHomeFile))
	if err := cmd.MarkFlagFilename("output"); err != nil {
		l.Log().Fatalln("Failed to mark flag --output as filename")
	}
	cmd.Flags().BoolVarP(&mergeKubeconfigFlags.targetDefault, "kubeconfig-merge-default", "d", false, fmt.Sprintf("Merge into the default kubeconfig ($KUBECONFIG or %s)", clientcmd.RecommendedHomeFile))
	cmd.Flags().BoolVarP(&writeKubeConfigOptions.UpdateExisting, "update", "u", true, "Update conflicting fields in existing kubeconfig")
	cmd.Flags().BoolVarP(&writeKubeConfigOptions.UpdateCurrentContext, "kubeconfig-switch-context", "s", true, "Switch to new context")
	cmd.Flags().BoolVar(&writeKubeConfigOptions.OverwriteExisting, "overwrite", false, "[Careful!] Overwrite existing file, ignoring its contents")
	cmd.Flags().BoolVarP(&mergeKubeconfigFlags.all, "all", "a", false, "Get kubeconfigs from all existing clusters")

	// done
	return cmd
}
