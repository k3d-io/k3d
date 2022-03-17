/*
Copyright © 2020-2022 The k3d Author(s)

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
package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/plugin"
	k3dPlugin "github.com/k3d-io/k3d/v5/pkg/plugin"
	tabwriter "github.com/liggitt/tabwriter"
	"github.com/spf13/cobra"
)

type pluginListFlags struct {
	noHeader bool
}

func NewCmdPluginList() *cobra.Command {
	pluginListFlags := pluginListFlags{}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "get"},
		Short:   "List plugin(s)",
		Long:    "List plugin(s)",
		Args:    cobra.MinimumNArgs(0), // 0 or more; 0 = all
		Run: func(cmd *cobra.Command, args []string) {
			userHomeDir, err := os.UserHomeDir()
			if err != nil {
				l.Log().Fatalln(err)
			}

			// TODO: Override this variables from cobra/viper or using env vars
			pluginPath := filepath.Join(userHomeDir, ".k3d", "plugins")
			manifestName := plugin.DefaultManifestName

			plugins, err := k3dPlugin.LoadAll(pluginPath, manifestName)
			if err != nil {
				l.Log().Fatalln(err)
			}

			PrintPlugins(plugins, pluginListFlags)
		},
	}

	// add subcommands

	// add flags
	cmd.Flags().BoolVar(&pluginListFlags.noHeader, "no-headers", false, "Disable headers")

	return cmd
}

func PrintPlugins(plugins []*k3dPlugin.Plugin, flags pluginListFlags) {
	headers := &[]string{}
	if !flags.noHeader {
		headers = &[]string{"NAME", "VERSION", "DESCRIPTION"}
	}

	tabwriter := tabwriter.NewWriter(os.Stdout, 6, 4, 3, ' ', tabwriter.RememberWidths)
	defer tabwriter.Flush()

	_, err := fmt.Fprintf(tabwriter, "%s\n", strings.Join(*headers, "\t"))
	if err != nil {
		l.Log().Fatalln("Failed to print headers")
	}

	for _, plugin := range plugins {
		fmt.Fprintf(tabwriter, "%s\t%s\t%s\n", plugin.Manifest.Name, plugin.Manifest.Version, plugin.Manifest.ShortHelpMessage)
	}
}
