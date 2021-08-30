/*
Copyright © 2020-2021 The k3d Author(s)

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
	"bufio"
	"log"
	"os"
	"path"

	"github.com/rancher/k3d/v4/cmd/util"
	l "github.com/rancher/k3d/v4/pkg/logger"
	utils "github.com/rancher/k3d/v4/pkg/util"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// NewCmdPluginInstall returns a new cobra command
func NewCmdPluginInstall() *cobra.Command {

	// create new cobra command
	cmd := &cobra.Command{
		Use:   "install PLUGIN [PLUGIN...]",
		Short: "Install a plugin",
		Long: `Install a plugin

Examples:
  To install one plugin, run:
    k3d plugin install user/plugin

  To install the specific version of a plugin, use:
    k3d plugin install user/plugin@v0.0.1

  If you have a list of plugins in a file, run:
    k3d plugin install < plugins.txt

Remarks:
  If a plugin is already installed, it will be overridden.
`,
		Run: func(cmd *cobra.Command, args []string) {
			warnIfNotATerminal()
			printHelpIfNoArgs(cmd, args)

			plugins := getPlugins(args)

			// Get the path of the plugin folder
			pluginDir, err := utils.GetPluginDirOrCreate()
			if err != nil {
				l.Log().Fatal(err)
			}

			// Install all plugins
			for _, plugin := range plugins {
				// Get the plugin path
				pluginPath := path.Join(pluginDir, plugin.Name)

				// Download the plugin
				l.Log().Infof("Installing plugin %s", plugin.Name)
				err = util.DownloadPlugin(*plugin, pluginPath)
				if err != nil {
					l.Log().Errorf("Unable to download %s@%s", plugin.Name, plugin.Version)
					l.Log().Fatal(err)
				}

				l.Log().Debug("Changing file permissions")
				if err = os.Chmod(pluginPath, 0744); err != nil {
					l.Log().Errorf("Error while changing file permissions: %s", err)
				}

				l.Log().Infof("Plugin %s installed successfully", plugin.Name)
			}
		},
	}

	// add subcommands

	// add flags

	// done
	return cmd
}

// getPlugins reads plugins from the stdin if it is a file descriptor
// or from command args
func getPlugins(pluginNames []string) []*util.Plugin {
	// Ignore args if adding plugins using stdin
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		pluginNames = readPluginsFromStdin()
	}
	plugins := parsePlugins(pluginNames)

	return plugins
}

// parsePlugins reads a list of plugin names and returns the list of corresponding Plugins
func parsePlugins(pluginNames []string) []*util.Plugin {
	var plugins = make([]*util.Plugin, len(pluginNames))

	// Read plugins from args
	for index, pluginName := range pluginNames {
		plugin, err := util.NewPlugin(pluginName)
		if err != nil {
			log.Fatal(err)
		}
		plugins[index] = plugin
	}

	return plugins
}

// readPluginsFromStdin reads plugin names from os.Stdin.
// Returns the list of plugin names.
func readPluginsFromStdin() []string {
	var pluginNames []string

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		if pluginName := scanner.Text(); pluginName != "" {
			pluginNames = append(pluginNames, pluginName)
		}
	}

	return pluginNames
}

// Log a warning if the stdin is not a terminal
func warnIfNotATerminal() {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		l.Log().Warn("Stdin detected")
	}
}

// Show help and exit if k3d is launched in a terminal and there are 0 args
func printHelpIfNoArgs(cmd *cobra.Command, args []string) {
	if term.IsTerminal(int(os.Stdin.Fd())) && len(args) == 0 {
		if err := cmd.Help(); err != nil {
			l.Log().Errorln("Couldn't get help text")
			l.Log().Fatalln(err)
		}
	}
}
