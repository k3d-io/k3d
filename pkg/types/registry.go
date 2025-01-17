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
package types

// Registry Defaults
const (
	DefaultRegistryPort       = "5000"
	DefaultRegistryName       = DefaultObjectNamePrefix + "-registry"
	DefaultRegistriesFilePath = "/etc/rancher/k3s/registries.yaml"
	DefaultRegistryMountPath  = "/var/lib/registry"
	DefaultDockerHubAddress   = "registry-1.docker.io"
	// Default temporary path for the LocalRegistryHosting configmap, from where it will be applied via kubectl
	DefaultLocalRegistryHostingConfigmapTempPath = "/tmp/localRegistryHostingCM.yaml"
)

type RegistryOptions struct {
	ConfigFile    string        `json:"configFile,omitempty"`
	Proxy         RegistryProxy `json:"proxy,omitempty"`
	DeleteEnabled bool          `json:"deleteEnabled,omitempty"`
}

type RegistryProxy struct {
	RemoteURL string `json:"remoteURL"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
}

// Registry describes a k3d-managed registry
type Registry struct {
	ClusterRef   string          // filled automatically -> if created with a cluster
	Protocol     string          `json:"protocol,omitempty"` // default: http
	Host         string          `json:"host"`
	Image        string          `json:"image,omitempty"`
	Network      string          `json:"Network,omitempty"`
	Volumes      []string        `json:"Volumes,omitempty"`
	ExposureOpts ExposureOpts    `json:"expose"`
	Options      RegistryOptions `json:"options,omitempty"`
}

// RegistryExternal describes a minimal spec for an "external" registry
// "external" meaning, that it's unrelated to the current cluster
// e.g. used for the --registry-use flag registry reference
type RegistryExternal struct {
	Protocol string `json:"protocol,omitempty"` // default: http
	Host     string `json:"host"`
	Port     string `json:"port"`
}
