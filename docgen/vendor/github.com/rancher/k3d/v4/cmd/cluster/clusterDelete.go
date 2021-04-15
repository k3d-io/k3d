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
package cluster

import (
	"fmt"
	"os"
	"path"

	"github.com/rancher/k3d/v4/cmd/util"
	"github.com/rancher/k3d/v4/pkg/client"
	"github.com/rancher/k3d/v4/pkg/runtimes"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	k3dutil "github.com/rancher/k3d/v4/pkg/util"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

// NewCmdClusterDelete returns a new cobra command
func NewCmdClusterDelete() *cobra.Command {

	// create new cobra command
	cmd := &cobra.Command{
		Use:               "delete [NAME [NAME ...] | --all]",
		Aliases:           []string{"del", "rm"},
		Short:             "Delete cluster(s).",
		Long:              `Delete cluster(s).`,
		Args:              cobra.MinimumNArgs(0), // 0 or n arguments; 0 = default cluster name
		ValidArgsFunction: util.ValidArgsAvailableClusters,
		Run: func(cmd *cobra.Command, args []string) {
			clusters := parseDeleteClusterCmd(cmd, args)

			if len(clusters) == 0 {
				log.Infoln("No clusters found")
			} else {
				for _, c := range clusters {
					if err := client.ClusterDelete(cmd.Context(), runtimes.SelectedRuntime, c, k3d.ClusterDeleteOpts{SkipRegistryCheck: false}); err != nil {
						log.Fatalln(err)
					}
					log.Infoln("Removing cluster details from default kubeconfig...")
					if err := client.KubeconfigRemoveClusterFromDefaultConfig(cmd.Context(), c); err != nil {
						log.Warnln("Failed to remove cluster details from default kubeconfig")
						log.Warnln(err)
					}
					log.Infoln("Removing standalone kubeconfig file (if there is one)...")
					configDir, err := k3dutil.GetConfigDirOrCreate()
					if err != nil {
						log.Warnf("Failed to delete kubeconfig file: %+v", err)
					} else {
						kubeconfigfile := path.Join(configDir, fmt.Sprintf("kubeconfig-%s.yaml", c.Name))
						if err := os.Remove(kubeconfigfile); err != nil {
							if !os.IsNotExist(err) {
								log.Warnf("Failed to delete kubeconfig file '%s'", kubeconfigfile)
							}
						}
					}

					log.Infof("Successfully deleted cluster %s!", c.Name)
				}
			}

		},
	}

	// add subcommands

	// add flags
	cmd.Flags().BoolP("all", "a", false, "Delete all existing clusters")

	// done
	return cmd
}

// parseDeleteClusterCmd parses the command input into variables required to delete clusters
func parseDeleteClusterCmd(cmd *cobra.Command, args []string) []*k3d.Cluster {

	// --all
	var clusters []*k3d.Cluster

	if all, err := cmd.Flags().GetBool("all"); err != nil {
		log.Fatalln(err)
	} else if all {
		log.Infoln("Deleting all clusters...")
		clusters, err = client.ClusterList(cmd.Context(), runtimes.SelectedRuntime)
		if err != nil {
			log.Fatalln(err)
		}
		return clusters
	}

	clusternames := []string{k3d.DefaultClusterName}
	if len(args) != 0 {
		clusternames = args
	}

	for _, name := range clusternames {
		c, err := client.ClusterGet(cmd.Context(), runtimes.SelectedRuntime, &k3d.Cluster{Name: name})
		if err != nil {
			if err == client.ClusterGetNoNodesFoundError {
				continue
			}
			log.Fatalln(err)
		}
		clusters = append(clusters, c)
	}

	return clusters
}
