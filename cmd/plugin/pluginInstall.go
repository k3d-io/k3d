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
	"os"
	"path"
	"strings"

	"github.com/rancher/k3d/v4/cmd/util"
	utils "github.com/rancher/k3d/v4/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// parsePlugin parses the plugin name to extract the name and the version.
// If no version is specified, latest will be returned.
//
// parsePlugin expects plugin to be formatted as user/repo@version
func parsePlugin(plugin string) (string, string) {
	parsed := strings.Split(plugin, "@")
	name := parsed[0]

	// Version is not specified, using latest
	if len(parsed) == 1 {
		return name, "latest"
	}
	// Version in specified, returning it
	return name, parsed[1]
}

// NewCmdPluginInstall returns a new cobra command
func NewCmdPluginInstall() *cobra.Command {

	// create new cobra command
	cmd := &cobra.Command{
		Use:   "install PLUGIN",
		Short: "Install a plugin",
		Long:  `Install a plugin`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Parse plugin name and version
			repoName, pluginVersion := parsePlugin(args[0])

			// Get the path of the plugin folder
			pluginDir, err := utils.GetPluginDirOrCreate()
			if err != nil {
				log.Fatal(err)
			}

			// Get the plugin path
			pluginName := strings.Split(repoName, "/")[1]
			pluginPath := path.Join(pluginDir, pluginName)

			// Download the plugin
			log.Infof("Installing plugin %s", pluginName)
			err = util.DownloadPlugin(repoName, pluginVersion, pluginPath)
			if err != nil {
				log.Errorf("Unable to download %s@%s", repoName, pluginVersion)
				log.Fatal(err)
			}

			log.Debug("Changing file permissions")
			if err = os.Chmod(pluginPath, 0744); err != nil {
				log.Errorf("Error while changing file permissions: %s", err)
			}

			log.Infof("Plugin %s installed successfully", pluginName)
		},
	}

	// add subcommands

	// add flags

	// done
	return cmd
}
