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
package image

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/k3d-io/k3d/v5/cmd/util"
	"github.com/k3d-io/k3d/v5/pkg/client"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

// NewCmdImageImport returns a new cobra command
func NewCmdImageImport() *cobra.Command {
	loadImageOpts := k3d.ImageImportOpts{}

	// create new command
	cmd := &cobra.Command{
		Use:   "import [IMAGE | ARCHIVE [IMAGE | ARCHIVE...]]",
		Short: "Import image(s) from docker into k3d cluster(s).",
		Long: `Import image(s) from docker into k3d cluster(s).

If an IMAGE starts with the prefix 'docker.io/', then this prefix is stripped internally.
That is, 'docker.io/k3d-io/k3d-tools:latest' is treated as 'k3d-io/k3d-tools:latest'.

If an IMAGE starts with the prefix 'library/' (or 'docker.io/library/'), then this prefix is stripped internally.
That is, 'library/busybox:latest' (or 'docker.io/library/busybox:latest') are treated as 'busybox:latest'.

If an IMAGE does not have a version tag, then ':latest' is assumed.
That is, 'k3d-io/k3d-tools' is treated as 'k3d-io/k3d-tools:latest'.

A file ARCHIVE always takes precedence.
So if a file './k3d-io/k3d-tools' exists, k3d will try to import it instead of the IMAGE of the same name.`,
		Aliases: []string{"load"},
		Args:    cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			images, clusters := parseLoadImageCmd(cmd, args)

			loadModeStr, err := cmd.Flags().GetString("mode")
			if err != nil {
				l.Log().Errorln("No load-mode specified")
				l.Log().Fatalln(err)
			}
			if mode, ok := k3d.ImportModes[loadModeStr]; !ok {
				l.Log().Fatalf("Unknown image loading mode '%s'\n", loadModeStr)
			} else {
				loadImageOpts.Mode = mode
			}

			l.Log().Debugf("Importing image(s) [%+v] from runtime [%s] into cluster(s) [%+v]...", images, runtimes.SelectedRuntime, clusters)
			errOccurred := false
			for _, cluster := range clusters {
				l.Log().Infof("Importing image(s) into cluster '%s'", cluster.Name)
				if err := client.ImageImportIntoClusterMulti(cmd.Context(), runtimes.SelectedRuntime, images, cluster, loadImageOpts); err != nil {
					l.Log().Errorf("Failed to import image(s) into cluster '%s': %+v", cluster.Name, err)
					errOccurred = true
				}
			}
			if errOccurred {
				l.Log().Warnln("At least one error occured while trying to import the image(s) into the selected cluster(s)")
				os.Exit(1)
			}
			l.Log().Infof("Successfully imported %d image(s) into %d cluster(s)", len(images), len(clusters))
		},
	}

	/*********
	 * Flags *
	 *********/
	cmd.Flags().StringArrayP("cluster", "c", []string{k3d.DefaultClusterName}, "Select clusters to load the image to.")
	if err := cmd.RegisterFlagCompletionFunc("cluster", util.ValidArgsAvailableClusters); err != nil {
		l.Log().Fatalln("Failed to register flag completion for '--cluster'", err)
	}

	cmd.Flags().BoolVarP(&loadImageOpts.KeepTar, "keep-tarball", "k", false, "Do not delete the tarball containing the saved images from the shared volume")
	cmd.Flags().BoolVarP(&loadImageOpts.KeepToolsNode, "keep-tools", "t", false, "Do not delete the tools node after import")
	cmd.Flags().StringP("mode", "m", string(k3d.ImportModeToolsNode), "Which method to use to import images into the cluster [auto, direct, tools]. See https://k3d.io/stable/usage/importing_images/")
	/* Subcommands */

	// done
	return cmd
}

// parseLoadImageCmd parses the command input into variables required to create a cluster
func parseLoadImageCmd(cmd *cobra.Command, args []string) ([]string, []*k3d.Cluster) {
	// --cluster
	clusterNames, err := cmd.Flags().GetStringArray("cluster")
	if err != nil {
		l.Log().Fatalln(err)
	}
	clusters := []*k3d.Cluster{}
	for _, clusterName := range clusterNames {
		cluster, err := client.ClusterGet(cmd.Context(), runtimes.SelectedRuntime, &k3d.Cluster{Name: clusterName})
		if err != nil {
			l.Log().Fatalf("failed to get cluster %s: %v", clusterName, err)
		}
		clusters = append(clusters, cluster)
	}

	// images
	images := args
	if len(images) == 0 {
		l.Log().Fatalln("No images specified!")
	}

	return images, clusters
}
