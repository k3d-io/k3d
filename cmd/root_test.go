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

package cmd

import (
	"testing"
)

func Test_runtimeInitRequired(t *testing.T) {
	tests := map[string]struct {
		givenArgs        []string
		expectedRequired bool
	}{
		"version does not need a container runtime": {
			givenArgs:        []string{"version"},
			expectedRequired: false,
		},
		"version list only queries image registries": {
			givenArgs:        []string{"version", "list", "k3s"},
			expectedRequired: false,
		},
		"completion is generated from the command tree alone": {
			givenArgs:        []string{"completion", "bash"},
			expectedRequired: false,
		},
		"cluster create needs a container runtime": {
			givenArgs:        []string{"cluster", "create"},
			expectedRequired: true,
		},
		"node list needs a container runtime": {
			givenArgs:        []string{"node", "list"},
			expectedRequired: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rootCmd := NewCmdK3d()

			cmd, _, err := rootCmd.Find(tt.givenArgs)
			if err != nil {
				t.Fatalf("failed to find command %v: %v", tt.givenArgs, err)
			}

			if actual := runtimeInitRequired(cmd); actual != tt.expectedRequired {
				t.Errorf("runtimeInitRequired(%v) = %v, expected %v", tt.givenArgs, actual, tt.expectedRequired)
			}
		})
	}
}

func Test_runtimeInitRequired_isFalseForRootCommand(t *testing.T) {
	// `k3d` and `k3d --version` only print usage / the version, so they must not need a runtime either
	rootCmd := NewCmdK3d()

	if runtimeInitRequired(rootCmd) {
		t.Error("runtimeInitRequired(rootCmd) = true, expected false")
	}
}
