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

package node

import (
	"os"

	"github.com/k3d-io/k3d/v5/cmd/util"
	"github.com/k3d-io/k3d/v5/pkg/client"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/spf13/cobra"
)

var exportPath string

func NewCmdNodeExportLogs() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "export-logs NODE",
		Short:             "Export logs of a k3d node",
		Long:              `Export logs of a k3d node.`,
		ValidArgsFunction: util.ValidArgsAvailableNodes,
		Run: func(cmd *cobra.Command, args []string) {
			node := parseExportLogsCmd(cmd, args)
			l.Log().Fatalln(runtimes.SelectedRuntime.ExportLogsFromNode(cmd.Context(), node, exportPath))
		},
	}

	cwd, err := os.Getwd()
	if err != nil {
		l.Log().Fatalln(err)
	}
	cmd.Flags().StringVarP(&exportPath, "path", "p", cwd, "Path to export the logs to")

	return cmd
}

func parseExportLogsCmd(cmd *cobra.Command, args []string) *k3d.Node {
	if len(args) == 0 || len(args[0]) == 0 {
		l.Log().Fatalln("No node name given")
	}

	node, err := client.NodeGet(cmd.Context(), runtimes.SelectedRuntime, &k3d.Node{Name: args[0]})
	if err != nil {
		l.Log().Fatalln(err)
	}
	return node
}
