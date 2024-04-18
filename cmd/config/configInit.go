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
package config

import (
	"fmt"
	"os"

	config "github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/spf13/cobra"
)

// NewCmdConfigInit returns a new cobra command
func NewCmdConfigInit() *cobra.Command {
	var output string
	var force bool

	cmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{"create"},
		Run: func(cmd *cobra.Command, args []string) {
			l.Log().Infoln("COMING SOON: print a basic k3d config with default pre-filled.")
			if output == "-" {
				fmt.Println(config.DefaultConfig)
			} else {
				// check if file exists
				var file *os.File
				var err error
				_, err = os.Stat(output)
				if os.IsNotExist(err) || force {
					// create/overwrite file
					file, err = os.Create(output)
					if err != nil {
						l.Log().Fatalf("Failed to create/overwrite output file: %s", err)
					}
					defer file.Close()
					// write content
					if _, err = file.WriteString(config.DefaultConfig); err != nil {
						l.Log().Fatalf("Failed to write to output file: %+v", err)
					}
				} else if err != nil {
					l.Log().Fatalf("Failed to stat output file: %+v", err)
				} else {
					l.Log().Errorln("Output file exists and --force was not set")
					os.Exit(1)
				}
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "k3d-default.yaml", "Write a default k3d config")
	if err := cmd.MarkFlagFilename("output", "yaml", "yml"); err != nil {
		l.Log().Fatalf("Failed to mark flag 'output' as filename flag: %v", err)
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force overwrite of target file")

	return cmd
}
