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
package plugin

import (
	"io/ioutil"
	"path/filepath"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// listSubdirs returns a list of folders inside basepath
func listSubdirs(basepath string) ([]string, error) {
	globPath := filepath.Join(basepath, "*")
	subdirs, err := filepath.Glob(globPath)
	if err != nil {
		return nil, errors.Wrapf(err, "Error while listing subdirectories of %s", basepath)
	}

	return subdirs, nil
}

// LoadOne loads the plugin located in manifestPath
func LoadOne(manifestPath string) (*Plugin, error) {
	data, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return nil, errors.Wrapf(err, "Error while loading %s plugin", manifestPath)
	}

	plugin := &Plugin{}

	// Load plugin manifest
	err = yaml.UnmarshalStrict(data, &plugin.Manifest)
	if err != nil {
		return nil, errors.Wrapf(err, "Error while unmarshalling plugin at %s", manifestPath)
	}
	// Set plugin directory
	plugin.Directory = filepath.Dir(manifestPath)

	return plugin, nil
}

// LoadAll loads all plugins inside the specified pluginsPath
func LoadAll(pluginsPath, manifestName string) ([]*Plugin, error) {
	pluginDirectories, err := listSubdirs(pluginsPath)
	if err != nil {
		return nil, err
	}
	l.Log().Debugf("Read subdirectory of %s, found %d subdir(s)", pluginsPath, len(pluginDirectories))

	plugins := []*Plugin{}

	for _, dir := range pluginDirectories {
		manifestPath := filepath.Join(dir, manifestName)
		l.Log().Debugf("Loading %s", manifestPath)

		plugin, err := LoadOne(manifestPath)
		if err != nil {
			return nil, err
		}

		l.Log().Debugf("%s loaded", plugin.Manifest.Name)

		plugins = append(plugins, plugin)
	}

	return plugins, nil
}
