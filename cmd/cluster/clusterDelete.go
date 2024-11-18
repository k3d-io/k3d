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
package cluster

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/k3d-io/k3d/v5/cmd/util"
	cliconfig "github.com/k3d-io/k3d/v5/cmd/util/config"
	"github.com/k3d-io/k3d/v5/pkg/client"
	"github.com/k3d-io/k3d/v5/pkg/config"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	k3dutil "github.com/k3d-io/k3d/v5/pkg/util"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	clusterDeleteCfgViper = viper.New()
	clusterDeletePpViper  = viper.New()
)

func initClusterDeleteConfig() error {
	// Viper for pre-processed config options
	clusterDeletePpViper.SetEnvPrefix("K3D")
	clusterDeletePpViper.AutomaticEnv()
	clusterDeletePpViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if l.Log().GetLevel() >= logrus.DebugLevel {
		c, _ := yaml.Marshal(clusterDeletePpViper.AllSettings())
		l.Log().Debugf("Additional CLI Configuration:\n%s", c)
	}

	return cliconfig.InitViperWithConfigFile(clusterDeleteCfgViper, clusterDeletePpViper.GetString("config"))
}

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
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initClusterDeleteConfig()
		},
		Run: func(cmd *cobra.Command, args []string) {
			clusters := parseDeleteClusterCmd(cmd, args)

			if len(clusters) == 0 {
				l.Log().Infoln("No clusters found")
			} else {
				for _, c := range clusters {
					if err := client.ClusterDelete(cmd.Context(), runtimes.SelectedRuntime, c, k3d.ClusterDeleteOpts{SkipRegistryCheck: false}); err != nil {
						l.Log().Fatalln(err)
					}
					l.Log().Infoln("Removing cluster details from default kubeconfig...")
					if err := client.KubeconfigRemoveClusterFromDefaultConfig(cmd.Context(), c); err != nil {
						l.Log().Warnln("Failed to remove cluster details from default kubeconfig")
						l.Log().Warnln(err)
					}
					l.Log().Infoln("Removing standalone kubeconfig file (if there is one)...")
					configDir, err := k3dutil.GetConfigDirOrCreate()
					if err != nil {
						l.Log().Warnf("Failed to delete kubeconfig file: %+v", err)
					} else {
						kubeconfigfile := path.Join(configDir, fmt.Sprintf("kubeconfig-%s.yaml", c.Name))
						if err := os.Remove(kubeconfigfile); err != nil {
							if !os.IsNotExist(err) {
								l.Log().Warnf("Failed to delete kubeconfig file '%s'", kubeconfigfile)
							}
						}
					}

					l.Log().Infof("Successfully deleted cluster %s!", c.Name)
				}
			}
		},
	}

	// add subcommands

	// add flags
	cmd.Flags().BoolP("all", "a", false, "Delete all existing clusters")

	/***************
	 * Config File *
	 ***************/

	cmd.Flags().StringP("config", "c", "", "Path of a config file to use")
	_ = clusterDeletePpViper.BindPFlag("config", cmd.Flags().Lookup("config"))
	if err := cmd.MarkFlagFilename("config", "yaml", "yml"); err != nil {
		l.Log().Fatalln("Failed to mark flag 'config' as filename flag")
	}

	// done
	return cmd
}

// parseDeleteClusterCmd parses the command input into variables required to delete clusters
func parseDeleteClusterCmd(cmd *cobra.Command, args []string) []*k3d.Cluster {
	// --all
	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		l.Log().Fatalln(err)
	}

	// --all was set
	if all {
		l.Log().Infoln("Deleting all clusters...")
		clusters, err := client.ClusterList(cmd.Context(), runtimes.SelectedRuntime)
		if err != nil {
			l.Log().Fatalln(err)
		}
		return clusters
	}

	// args
	if len(args) != 0 {
		return getClusters(cmd.Context(), args...)
	}

	// --config
	if clusterDeletePpViper.GetString("config") != "" {
		cfg, err := config.SimpleConfigFromViper(clusterDeleteCfgViper)
		if err != nil {
			l.Log().Fatalln(err)
		}
		if cfg.Name != "" {
			return getClusters(cmd.Context(), cfg.Name)
		}
	}

	// default
	return getClusters(cmd.Context(), k3d.DefaultClusterName)
}

func getClusters(ctx context.Context, clusternames ...string) []*k3d.Cluster {
	var clusters []*k3d.Cluster
	for _, name := range clusternames {
		c, err := client.ClusterGet(ctx, runtimes.SelectedRuntime, &k3d.Cluster{Name: name})
		if err != nil {
			if errors.Is(err, client.ClusterGetNoNodesFoundError) {
				l.Log().Infof("No nodes found for cluster '%s', nothing to delete.", name)
				continue
			}
			l.Log().Fatalln(err)
		}
		clusters = append(clusters, c)
	}
	return clusters
}
