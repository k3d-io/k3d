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
package util

import (
	"os"
	"path/filepath"
	"testing"

	homedir "github.com/mitchellh/go-homedir"
	"gotest.tools/assert"
)

func TestGetConfigDirOrCreate(t *testing.T) {
	testSets := map[string]struct {
		xdgConfigHome func(tmpDir string) string
		unset         bool
		expected      func(tmpDir string) string
	}{
		"XDG_CONFIG_HOME set": {
			xdgConfigHome: func(tmpDir string) string { return filepath.Join(tmpDir, "xdg") },
			expected:      func(tmpDir string) string { return filepath.Join(tmpDir, "xdg", "k3d") },
		},
		"XDG_CONFIG_HOME set with trailing slash": {
			xdgConfigHome: func(tmpDir string) string { return filepath.Join(tmpDir, "xdg") + "/" },
			expected:      func(tmpDir string) string { return filepath.Join(tmpDir, "xdg", "k3d") },
		},
		"XDG_CONFIG_HOME unset": {
			xdgConfigHome: func(tmpDir string) string { return "" },
			unset:         true,
			expected:      func(tmpDir string) string { return filepath.Join(tmpDir, "home", ".config", "k3d") },
		},
		"XDG_CONFIG_HOME empty": {
			xdgConfigHome: func(tmpDir string) string { return "" },
			expected:      func(tmpDir string) string { return filepath.Join(tmpDir, "home", ".config", "k3d") },
		},
	}
	for name, testSet := range testSets {
		t.Run(name, func(t *testing.T) {
			tmpDir := useTempHome(t)
			t.Setenv("XDG_CONFIG_HOME", testSet.xdgConfigHome(tmpDir)) // also restores the original value after the test
			if testSet.unset {
				assert.NilError(t, os.Unsetenv("XDG_CONFIG_HOME"))
			}

			actual, err := GetConfigDirOrCreate()
			assert.NilError(t, err)
			assert.Equal(t, testSet.expected(tmpDir), actual)

			info, err := os.Stat(actual)
			assert.NilError(t, err)
			assert.Assert(t, info.IsDir())
		})
	}
}

// useTempHome sets $HOME to a new directory below a temporary directory and returns the path of the temporary directory
func useTempHome(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmpDir, "home"))
	// go-homedir caches the home directory
	homedir.Reset()
	t.Cleanup(homedir.Reset)
	return tmpDir
}
