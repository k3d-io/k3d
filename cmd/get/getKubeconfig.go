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

	k3dutil "github.com/rancher/k3d/cmd/util"
	k3dcluster "github.com/rancher/k3d/pkg/cluster"
	"github.com/rancher/k3d/pkg/runtimes"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	log "github.com/sirupsen/logrus"
)

// TODO : deal with --all flag to manage differentiate started cluster and stopped cluster like `docker ps` and `docker ps -a`
type getKubeconfigFlags struct {
	output string
}

// NewCmdGetKubeconfig returns a new cobra command
func NewCmdGetKubeconfig() *cobra.Command {

	writeKubeConfigOptions := k3dcluster.WriteKubeConfigOptions{}

	getKubeconfigFlags := getKubeconfigFlags{}

	// create new command
	cmd := &cobra.Command{
		Use:   "kubeconfig [CLUSTER [CLUSTER [...]] | --all]", // TODO: getKubeconfig: allow more than one cluster name or even --all
		Short: "Get kubeconfig",
		Long:  `Get kubeconfig.`,
		Run: func(cmd *cobra.Command, args []string) {

			// generate list of clusters
			clusters := k3dutil.BuildClusterList(cmd.Context(), args)

			// get kubeconfigs from all clusters
			errorGettingKubeconfig := false
			for _, cluster := range clusters {
				log.Debugf("Getting kubeconfig for cluster '%s'", cluster.Name)
				var err error
				getKubeconfigFlags.output, err = k3dcluster.GetAndWriteKubeConfig(cmd.Context(),
					runtimes.SelectedRuntime,
					cluster,
					getKubeconfigFlags.output,
					&writeKubeConfigOptions,
				)
				if err != nil {
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

	// done
	return cmd
}
