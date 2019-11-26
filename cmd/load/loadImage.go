/*
Copyright © 2019 Thorsten Klein <iwilltry42@gmail.com>

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
package load

import (
	"github.com/spf13/cobra"

	"github.com/rancher/k3d/pkg/runtimes"
	k3d "github.com/rancher/k3d/pkg/types"

	log "github.com/sirupsen/logrus"
)

// NewCmdLoadImage returns a new cobra command
func NewCmdLoadImage() *cobra.Command {

	// create new command
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Load an image from docker into a k3d cluster.",
		Long:  `Load an image from docker into a k3d cluster.`,
		Run: func(cmd *cobra.Command, args []string) {
			runtime, images, clusters := parseLoadImageCmd(cmd, args)
			log.Debugln("Load Images not yet implemented: ", runtime, images, clusters)
		},
	}

	/*********
	 * Flags *
	 *********/

	/* Subcommands */

	// done
	return cmd
}

// parseLoadImageCmd parses the command input into variables required to create a cluster
func parseLoadImageCmd(cmd *cobra.Command, args []string) (runtimes.Runtime, []string, []k3d.Cluster) {
	return nil, nil, nil
}
