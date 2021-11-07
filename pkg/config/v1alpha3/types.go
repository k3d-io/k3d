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

package v1alpha3

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	config "github.com/rancher/k3d/v5/pkg/config/types"
	k3d "github.com/rancher/k3d/v5/pkg/types"
	"github.com/rancher/k3d/v5/version"
)

const ApiVersion = "k3d.io/v1alpha3"

// JSONSchema describes the schema used to validate config files
//go:embed schema.json
var JSONSchema string

// DefaultConfigTpl for printing
const DefaultConfigTpl = `---
apiVersion: k3d.io/v1alpha3
kind: Simple
name: %s
servers: 1
agents: 0
image: %s
`

// DefaultConfig templated DefaultConfigTpl
var DefaultConfig = fmt.Sprintf(
	DefaultConfigTpl,
	k3d.DefaultClusterName,
	fmt.Sprintf("%s:%s", k3d.DefaultK3sImageRepo, version.K3sVersion),
)

type VolumeWithNodeFilters struct {
	Volume      string   `mapstructure:"volume" yaml:"volume,omitempty" json:"volume,omitempty"`
	NodeFilters []string `mapstructure:"nodeFilters" yaml:"nodeFilters,omitempty" json:"nodeFilters,omitempty"`
}

type PortWithNodeFilters struct {
	Port        string   `mapstructure:"port" yaml:"port,omitempty" json:"port,omitempty"`
	NodeFilters []string `mapstructure:"nodeFilters" yaml:"nodeFilters,omitempty" json:"nodeFilters,omitempty"`
}

type LabelWithNodeFilters struct {
	Label       string   `mapstructure:"label" yaml:"label,omitempty" json:"label,omitempty"`
	NodeFilters []string `mapstructure:"nodeFilters" yaml:"nodeFilters,omitempty" json:"nodeFilters,omitempty"`
}

type EnvVarWithNodeFilters struct {
	EnvVar      string   `mapstructure:"envVar" yaml:"envVar,omitempty" json:"envVar,omitempty"`
	NodeFilters []string `mapstructure:"nodeFilters" yaml:"nodeFilters,omitempty" json:"nodeFilters,omitempty"`
}

type K3sArgWithNodeFilters struct {
	Arg         string   `mapstructure:"arg" yaml:"arg,omitempty" json:"arg,omitempty"`
	NodeFilters []string `mapstructure:"nodeFilters" yaml:"nodeFilters,omitempty" json:"nodeFilters,omitempty"`
}

type SimpleConfigRegistryCreateConfig struct {
	Name     string `mapstructure:"name" yaml:"name,omitempty" json:"name,omitempty"`
	Host     string `mapstructure:"host" yaml:"host,omitempty" json:"host,omitempty"`
	HostPort string `mapstructure:"hostPort" yaml:"hostPort,omitempty" json:"hostPort,omitempty"`
}

// SimpleConfigOptionsKubeconfig describes the set of options referring to the kubeconfig during cluster creation.
type SimpleConfigOptionsKubeconfig struct {
	UpdateDefaultKubeconfig bool `mapstructure:"updateDefaultKubeconfig" yaml:"updateDefaultKubeconfig" json:"updateDefaultKubeconfig,omitempty"` // default: true
	SwitchCurrentContext    bool `mapstructure:"switchCurrentContext" yaml:"switchCurrentContext" json:"switchCurrentContext,omitempty"`          //nolint:lll    // default: true
}

type SimpleConfigOptions struct {
	K3dOptions        SimpleConfigOptionsK3d        `mapstructure:"k3d" yaml:"k3d" json:"k3d"`
	K3sOptions        SimpleConfigOptionsK3s        `mapstructure:"k3s" yaml:"k3s" json:"k3s"`
	KubeconfigOptions SimpleConfigOptionsKubeconfig `mapstructure:"kubeconfig" yaml:"kubeconfig" json:"kubeconfig"`
	Runtime           SimpleConfigOptionsRuntime    `mapstructure:"runtime" yaml:"runtime" json:"runtime"`
}

type SimpleConfigOptionsRuntime struct {
	GPURequest    string                 `mapstructure:"gpuRequest" yaml:"gpuRequest,omitempty" json:"gpuRequest,omitempty"`
	ServersMemory string                 `mapstructure:"serversMemory" yaml:"serversMemory,omitempty" json:"serversMemory,omitempty"`
	AgentsMemory  string                 `mapstructure:"agentsMemory" yaml:"agentsMemory,omitempty" json:"agentsMemory,omitempty"`
	Labels        []LabelWithNodeFilters `mapstructure:"labels" yaml:"labels,omitempty" json:"labels,omitempty"`
}

type SimpleConfigOptionsK3d struct {
	Wait                bool                               `mapstructure:"wait" yaml:"wait" json:"wait"`
	Timeout             time.Duration                      `mapstructure:"timeout" yaml:"timeout,omitempty" json:"timeout,omitempty"`
	DisableLoadbalancer bool                               `mapstructure:"disableLoadbalancer" yaml:"disableLoadbalancer" json:"disableLoadbalancer"`
	DisableImageVolume  bool                               `mapstructure:"disableImageVolume" yaml:"disableImageVolume" json:"disableImageVolume"`
	NoRollback          bool                               `mapstructure:"disableRollback" yaml:"disableRollback" json:"disableRollback"`
	NodeHookActions     []k3d.NodeHookAction               `mapstructure:"nodeHookActions" yaml:"nodeHookActions,omitempty" json:"nodeHookActions,omitempty"`
	Loadbalancer        SimpleConfigOptionsK3dLoadbalancer `mapstructure:"loadbalancer" yaml:"loadbalancer,omitempty" json:"loadbalancer,omitempty"`
}

type SimpleConfigOptionsK3dLoadbalancer struct {
	ConfigOverrides []string `mapstructure:"configOverrides" yaml:"configOverrides,omitempty" json:"configOverrides,omitempty"`
}

type SimpleConfigOptionsK3s struct {
	ExtraArgs  []K3sArgWithNodeFilters `mapstructure:"extraArgs" yaml:"extraArgs,omitempty" json:"extraArgs,omitempty"`
	NodeLabels []LabelWithNodeFilters  `mapstructure:"nodeLabels" yaml:"nodeLabels,omitempty" json:"nodeLabels,omitempty"`
}

type SimpleConfigRegistries struct {
	Use    []string                          `mapstructure:"use" yaml:"use,omitempty" json:"use,omitempty"`
	Create *SimpleConfigRegistryCreateConfig `mapstructure:"create" yaml:"create,omitempty" json:"create,omitempty"`
	Config string                            `mapstructure:"config" yaml:"config,omitempty" json:"config,omitempty"` // registries.yaml (k3s config for containerd registry override)
}

type SimpleConfigRegistriesIntermediateV1alpha2 struct {
	Use []string `mapstructure:"use" yaml:"use,omitempty" json:"use,omitempty"`
	// Field "Create" changed significantly, so it's dropped here
	Config string `mapstructure:"config" yaml:"config,omitempty" json:"config,omitempty"` // registries.yaml (k3s config for containerd registry override)
}

// SimpleConfig describes the toplevel k3d configuration file.
type SimpleConfig struct {
	config.TypeMeta `mapstructure:",squash" yaml:",inline"`
	Name            string                  `mapstructure:"name" yaml:"name,omitempty" json:"name,omitempty"`
	Servers         int                     `mapstructure:"servers" yaml:"servers,omitempty" json:"servers,omitempty"` //nolint:lll    // default 1
	Agents          int                     `mapstructure:"agents" yaml:"agents,omitempty" json:"agents,omitempty"`    //nolint:lll    // default 0
	ExposeAPI       SimpleExposureOpts      `mapstructure:"kubeAPI" yaml:"kubeAPI,omitempty" json:"kubeAPI,omitempty"`
	Image           string                  `mapstructure:"image" yaml:"image,omitempty" json:"image,omitempty"`
	Network         string                  `mapstructure:"network" yaml:"network,omitempty" json:"network,omitempty"`
	Subnet          string                  `mapstructure:"subnet" yaml:"subnet,omitempty" json:"subnet,omitempty"`
	ClusterToken    string                  `mapstructure:"token" yaml:"clusterToken,omitempty" json:"clusterToken,omitempty"` // default: auto-generated
	Volumes         []VolumeWithNodeFilters `mapstructure:"volumes" yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Ports           []PortWithNodeFilters   `mapstructure:"ports" yaml:"ports,omitempty" json:"ports,omitempty"`
	Options         SimpleConfigOptions     `mapstructure:"options" yaml:"options,omitempty" json:"options,omitempty"`
	Env             []EnvVarWithNodeFilters `mapstructure:"env" yaml:"env,omitempty" json:"env,omitempty"`
	Registries      SimpleConfigRegistries  `mapstructure:"registries" yaml:"registries,omitempty" json:"registries,omitempty"`
}

type SimpleConfigIntermediateV1alpha2 struct {
	config.TypeMeta `mapstructure:",squash" yaml:",inline"`
	Name            string                                     `mapstructure:"name" yaml:"name,omitempty" json:"name,omitempty"`
	Servers         int                                        `mapstructure:"servers" yaml:"servers,omitempty" json:"servers,omitempty"` //nolint:lll    // default 1
	Agents          int                                        `mapstructure:"agents" yaml:"agents,omitempty" json:"agents,omitempty"`    //nolint:lll    // default 0
	ExposeAPI       SimpleExposureOpts                         `mapstructure:"kubeAPI" yaml:"kubeAPI,omitempty" json:"kubeAPI,omitempty"`
	Image           string                                     `mapstructure:"image" yaml:"image,omitempty" json:"image,omitempty"`
	Network         string                                     `mapstructure:"network" yaml:"network,omitempty" json:"network,omitempty"`
	Subnet          string                                     `mapstructure:"subnet" yaml:"subnet,omitempty" json:"subnet,omitempty"`
	ClusterToken    string                                     `mapstructure:"token" yaml:"clusterToken,omitempty" json:"clusterToken,omitempty"` // default: auto-generated
	Volumes         []VolumeWithNodeFilters                    `mapstructure:"volumes" yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Ports           []PortWithNodeFilters                      `mapstructure:"ports" yaml:"ports,omitempty" json:"ports,omitempty"`
	Options         SimpleConfigOptions                        `mapstructure:"options" yaml:"options,omitempty" json:"options,omitempty"`
	Env             []EnvVarWithNodeFilters                    `mapstructure:"env" yaml:"env,omitempty" json:"env,omitempty"`
	Registries      SimpleConfigRegistriesIntermediateV1alpha2 `mapstructure:"registries" yaml:"registries,omitempty" json:"registries,omitempty"`
}

// SimpleExposureOpts provides a simplified syntax compared to the original k3d.ExposureOpts
type SimpleExposureOpts struct {
	Host     string `mapstructure:"host" yaml:"host,omitempty" json:"host,omitempty"`
	HostIP   string `mapstructure:"hostIP" yaml:"hostIP,omitempty" json:"hostIP,omitempty"`
	HostPort string `mapstructure:"hostPort" yaml:"hostPort,omitempty" json:"hostPort,omitempty"`
}

// GetKind implements Config.GetKind
func (c SimpleConfig) GetKind() string {
	return "Simple"
}

func (c SimpleConfig) GetAPIVersion() string {
	return ApiVersion
}

// ClusterConfig describes a single cluster config
type ClusterConfig struct {
	config.TypeMeta   `mapstructure:",squash" yaml:",inline"`
	Cluster           k3d.Cluster                   `mapstructure:",squash" yaml:",inline"`
	ClusterCreateOpts k3d.ClusterCreateOpts         `mapstructure:"options" yaml:"options"`
	KubeconfigOpts    SimpleConfigOptionsKubeconfig `mapstructure:"kubeconfig" yaml:"kubeconfig"`
}

// GetKind implements Config.GetKind
func (c ClusterConfig) GetKind() string {
	return "Simple"
}

func (c ClusterConfig) GetAPIVersion() string {
	return ApiVersion
}

// ClusterListConfig describes a list of clusters
type ClusterListConfig struct {
	config.TypeMeta `mapstructure:",squash" yaml:",inline"`
	Clusters        []k3d.Cluster `mapstructure:"clusters" yaml:"clusters"`
}

func (c ClusterListConfig) GetKind() string {
	return "Simple"
}

func (c ClusterListConfig) GetAPIVersion() string {
	return ApiVersion
}

func GetConfigByKind(kind string) (config.Config, error) {

	// determine config kind
	switch strings.ToLower(kind) {
	case "simple":
		return SimpleConfig{}, nil
	case "cluster":
		return ClusterConfig{}, nil
	case "clusterlist":
		return ClusterListConfig{}, nil
	case "":
		return nil, fmt.Errorf("missing `kind` in config file")
	default:
		return nil, fmt.Errorf("unknown `kind` '%s' in config file", kind)
	}

}
