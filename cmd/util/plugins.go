/*
Copyright © 2020 The k3d Author(s)

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
package util

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func HandlePlugin(args []string) error {
	argsPrefix := []string{}

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue // drop flags
		}
		argsPrefix = append(argsPrefix, strings.ReplaceAll(arg, "-", "_")) // plugin executables assumed to have underscores
	}

	execPath := ""

	for len(argsPrefix) > 0 {
		path, found := FindPlugin(strings.Join(argsPrefix, "-"))

		if !found {
			argsPrefix = argsPrefix[:len(argsPrefix)-1] // drop last element
			continue
		}

		execPath = path
		break
	}

	if execPath == "" {
		return nil
	}

	return ExecPlugin(execPath, args[len(argsPrefix):], os.Environ())

}

// FindPlugin tries to find the plugin executable on the filesystem
func FindPlugin(name string) (string, bool) {
	path, err := exec.LookPath(fmt.Sprintf("%s-%s", k3d.DefaultObjectNamePrefix, name))
	if err == nil && len(path) > 0 {
		return path, true
	}
	return "", false
}

// ExecPlugin executes a found plugin
func ExecPlugin(path string, args []string, env []string) error {

	return nil
}
