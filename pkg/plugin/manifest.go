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

// DefaultManifestName is the name of the manifest file that describes the plugin
const DefaultManifestName = "k3d-plugin.yaml"

// Manifest PluginManifest describes plugin attributes
type Manifest struct {
	Name             string `yaml:"name"`
	Version          string `yaml:"version"`
	ShortHelpMessage string `yaml:"shortHelpMessage"`
	HelpMessage      string `yaml:"helpMessage"`
	Command          string `yaml:"command"`
}

type Plugin struct {
	Manifest  *Manifest
	Directory string
}
