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
package k3s

const (
	K3sPathStorage              = "/var/lib/rancher/k3s/storage"
	K3sPathManifests            = "/var/lib/rancher/k3s/server/manifests"
	K3sPathManifestsCustom      = "/var/lib/rancher/k3s/server/manifests/custom" // custom subfolder
	K3sPathContainerdConfig     = "/var/lib/rancher/k3s/agent/etc/containerd/config.toml"
	K3sPathContainerdConfigTmpl = "/var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl"
	K3sPathRegistryConfig       = "/etc/rancher/k3s/registries.yaml"
)

var K3sPathShortcuts = map[string]string{
	"k3s-storage":          K3sPathStorage,
	"k3s-manifests":        K3sPathManifests,
	"k3s-manifests-custom": K3sPathManifestsCustom,
	"k3s-containerd":       K3sPathContainerdConfig,
	"k3s-containerd-tmpl":  K3sPathContainerdConfigTmpl,
	"k3s-registry-config":  K3sPathRegistryConfig,
}
