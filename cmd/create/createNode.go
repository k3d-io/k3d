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
package create

import (
	"github.com/spf13/cobra"

	"github.com/rancher/k3d/pkg/cluster"
	"github.com/rancher/k3d/pkg/runtimes"
	k3d "github.com/rancher/k3d/pkg/types"
	log "github.com/sirupsen/logrus"
)

// NewCmdCreateNode returns a new cobra command
func NewCmdCreateNode() *cobra.Command {

	// create new command
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Create a new k3s node in docker",
		Long:  `Create a new containerized k3s node (k3s in docker).`,
		Args:  cobra.ExactArgs(1), // exactly one name accepted
		Run: func(cmd *cobra.Command, args []string) {
			log.Debugln("create node called")
			rt, err := cmd.Flags().GetString("runtime")
			if err != nil {
				log.Debugln("runtime not defined")
			}
			runtime, err := runtimes.GetRuntime(rt)
			if err != nil {
				log.Fatalf("Unsupported runtime '%s'", rt)
			}
			if err := cluster.CreateNode(&k3d.Node{Name: args[0]}, runtime); err != nil {
				log.Fatalln(err)
			}
		},
	}

	// add flags
	cmd.Flags().Int("replicas", 1, "Number of replicas of this node specification.")

	// done
	return cmd
}
