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
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	homedir "github.com/mitchellh/go-homedir"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/pkg/types/k3s"
	yaml "gopkg.in/yaml.v3"
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

// ReadFileSource reads the file source which is either embedded in the k3d config file or relative to it.
func ReadFileSource(configFile, source string) ([]byte, error) {
	sourceContent := &yaml.Node{}
	sourceContent.SetString(source)

	// If the source input is embedded in the config file, use it as it is.
	if sourceContent.Style == yaml.LiteralStyle || sourceContent.Style == yaml.FoldedStyle {
		l.Log().Debugf("read source from embedded file with content '%s'", sourceContent.Value)
		return []byte(sourceContent.Value), nil
	}

	// If the source input is referenced as an external file, read its content.
	sourceFilePath := filepath.Join(filepath.Dir(configFile), sourceContent.Value)
	fileInfo, err := os.Stat(sourceFilePath)
	if err == nil && !fileInfo.IsDir() {
		fileContent, err := os.ReadFile(sourceFilePath)
		if err != nil {
			return nil, fmt.Errorf("cannot read file: %s", sourceFilePath)
		}
		l.Log().Debugf("read source from external file '%s'", sourceFilePath)
		return fileContent, nil
	}

	return nil, fmt.Errorf("could resolve source file path: %s", sourceFilePath)
}

// ResolveFileDestination determines the file destination and resolves it if it has a magic shortcut.
func ResolveFileDestination(destPath string) (string, error) {
	// If the destination path is absolute, then use it as it is.
	if filepath.IsAbs(destPath) {
		l.Log().Debugf("resolved destination with absolute path '%s'", destPath)
		return destPath, nil
	}

	// If the destination path has a magic shortcut, then resolve it and use it in the path.
	destPathTree := strings.Split(destPath, string(os.PathSeparator))
	if shortcutPath, found := k3s.K3sPathShortcuts[destPathTree[0]]; found {
		destPathTree[0] = shortcutPath
		destPathResolved := filepath.Join(destPathTree...)
		l.Log().Debugf("resolved destination with magic shortcut path: '%s'", destPathResolved)
		return filepath.Join(destPathResolved), nil
	}

	return "", fmt.Errorf("destination can be only absolute path or starts with predefined shortcut path. Could not resolve destination file path: %s", destPath)
}
