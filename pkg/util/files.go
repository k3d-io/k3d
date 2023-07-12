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
package util

import (
	"fmt"
	"os"
	"path"

	homedir "github.com/mitchellh/go-homedir"

	"github.com/k3d-io/k3d/v5/pkg/types"
)

// GetConfigDirOrCreate will return the base path of the k3d config directory or create it if it doesn't exist yet
// k3d's config directory will be $XDG_CONFIG_HOME (Unix)
func GetConfigDirOrCreate() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if len(configDir) == 0 {
		// build the path
		homeDir, err := homedir.Dir()
		if err != nil {
			return "", fmt.Errorf("failed to get user's home directory: %w", err)
		}
		configDir = path.Join(homeDir, types.DefaultConfigDirName)
	}

	// create directories if necessary
	if err := createDirIfNotExists(configDir); err != nil {
		return "", fmt.Errorf("failed to create config directory '%s': %w", configDir, err)
	}

	return configDir, nil
}

// createDirIfNotExists checks for the existence of a directory and creates it along with all required parents if not.
// It returns an error if the directory (or parents) couldn't be created and nil if it worked fine or if the path already exists.
func createDirIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, os.ModePerm)
	}
	return nil
}
