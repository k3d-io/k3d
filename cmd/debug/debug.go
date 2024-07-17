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
package debug

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/k3d-io/k3d/v5/cmd/util"
	"github.com/k3d-io/k3d/v5/pkg/client"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	"github.com/k3d-io/k3d/v5/pkg/types"
)

var nodeName string
var exportPath string
var components []string

// NewCmdDebug returns a new cobra command
func NewCmdDebug() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "debug",
		Hidden: true,
		Short:  "Debug k3d cluster(s)",
		Long:   `Debug k3d cluster(s)`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				l.Log().Errorln("Couldn't get help text")
				l.Log().Fatalln(err)
			}
		},
	}

	cmd.AddCommand(NewCmdDebugLoadbalancer())
	cmd.AddCommand(NewCmdExportLogs())
	return cmd
}

func NewCmdExportLogs() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "export-logs CLUSTER",
		Short:             "Export logs of a k3d cluster",
		Long:              "Export logs of a k3d cluster for all nodes or selective nodes using filters",
		ValidArgsFunction: util.ValidArgsAvailableClusters,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 || len(args[0]) == 0 {
				l.Log().Fatalln("Cluster name is required")
			}
			cluster, err := client.ClusterGet(cmd.Context(), runtimes.SelectedRuntime, &types.Cluster{Name: args[0]})
			if err != nil {
				l.Log().Fatalln(err)
			}
			exportPath = filepath.Join(exportPath, fmt.Sprintf("debug-logs-%s", cluster.Name))
			for _, node := range cluster.Nodes {
				if nodeName == "" || nodeName == node.Name {
					if err := runtimes.SelectedRuntime.ExportLogsFromNode(cmd.Context(), node, exportPath, components); err != nil {
						l.Log().Fatalln(err)
					}
				}
			}
		},
	}

	cwd, err := os.Getwd()
	if err != nil {
		l.Log().Fatalln(err)
	}

	cmd.Flags().StringVarP(&nodeName, "node", "n", "", "Node name to export logs from")
	cmd.Flags().StringVarP(&exportPath, "path", "p", cwd, "Path to export the logs to")

	return cmd
}

func NewCmdDebugLoadbalancer() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "loadbalancer",
		Aliases: []string{"lb"},
		Short:   "Debug the loadbalancer",
		Long:    `Debug the loadbalancer`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				l.Log().Errorln("Couldn't get help text")
				l.Log().Fatalln(err)
			}
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:               "get-config CLUSTERNAME",
		Args:              cobra.ExactArgs(1), // cluster name
		ValidArgsFunction: util.ValidArgsAvailableClusters,
		Run: func(cmd *cobra.Command, args []string) {
			c, err := client.ClusterGet(cmd.Context(), runtimes.SelectedRuntime, &types.Cluster{Name: args[0]})
			if err != nil {
				l.Log().Fatalln(err)
			}

			lbconf, err := client.GetLoadbalancerConfig(cmd.Context(), runtimes.SelectedRuntime, c)
			if err != nil {
				l.Log().Fatalln(err)
			}
			yamlized, err := yaml.Marshal(lbconf)
			if err != nil {
				l.Log().Fatalln(err)
			}
			fmt.Println(string(yamlized))
		},
	})

	return cmd
}
