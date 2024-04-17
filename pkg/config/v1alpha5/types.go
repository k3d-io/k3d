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

package v1alpha5

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	config "github.com/k3d-io/k3d/v5/pkg/config/types"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/version"
)

const ApiVersion = "k3d.io/v1alpha5"

// JSONSchema describes the schema used to validate config files
//
//go:embed schema.json
var JSONSchema string

// DefaultConfigTpl for printing
const DefaultConfigTpl = `---
apiVersion: k3d.io/v1alpha5
kind: Simple
metadata:
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
	Volume      string   `mapstructure:"volume" json:"volume,omitempty"`
	NodeFilters []string `mapstructure:"nodeFilters" json:"nodeFilters,omitempty"`
}

type PortWithNodeFilters struct {
	Port        string   `mapstructure:"port" json:"port,omitempty"`
	NodeFilters []string `mapstructure:"nodeFilters" json:"nodeFilters,omitempty"`
}

type LabelWithNodeFilters struct {
	Label       string   `mapstructure:"label" json:"label,omitempty"`
	NodeFilters []string `mapstructure:"nodeFilters" json:"nodeFilters,omitempty"`
}

type EnvVarWithNodeFilters struct {
	EnvVar      string   `mapstructure:"envVar" json:"envVar,omitempty"`
	NodeFilters []string `mapstructure:"nodeFilters" json:"nodeFilters,omitempty"`
}

type K3sArgWithNodeFilters struct {
	Arg         string   `mapstructure:"arg" json:"arg,omitempty"`
	NodeFilters []string `mapstructure:"nodeFilters" json:"nodeFilters,omitempty"`
}

type FileWithNodeFilters struct {
	Source      string   `mapstructure:"source" json:"source,omitempty"`
	Destination string   `mapstructure:"destination" json:"destination,omitempty"`
	Description string   `mapstructure:"description" json:"description,omitempty"`
	NodeFilters []string `mapstructure:"nodeFilters" json:"nodeFilters,omitempty"`
}

type SimpleConfigRegistryCreateConfig struct {
	Name     string            `mapstructure:"name" json:"name,omitempty"`
	Host     string            `mapstructure:"host" json:"host,omitempty"`
	HostPort string            `mapstructure:"hostPort" json:"hostPort,omitempty"`
	Image    string            `mapstructure:"image" json:"image,omitempty"`
	Proxy    k3d.RegistryProxy `mapstructure:"proxy" json:"proxy,omitempty"`
	Volumes  []string          `mapstructure:"volumes" json:"volumes,omitempty"`
}

// SimpleConfigOptionsKubeconfig describes the set of options referring to the kubeconfig during cluster creation.
type SimpleConfigOptionsKubeconfig struct {
	UpdateDefaultKubeconfig bool `mapstructure:"updateDefaultKubeconfig" json:"updateDefaultKubeconfig,omitempty"` // default: true
	SwitchCurrentContext    bool `mapstructure:"switchCurrentContext" json:"switchCurrentContext,omitempty"`       //nolint:lll    // default: true
}

type SimpleConfigOptions struct {
	K3dOptions        SimpleConfigOptionsK3d        `mapstructure:"k3d" json:"k3d"`
	K3sOptions        SimpleConfigOptionsK3s        `mapstructure:"k3s" json:"k3s"`
	KubeconfigOptions SimpleConfigOptionsKubeconfig `mapstructure:"kubeconfig" json:"kubeconfig"`
	Runtime           SimpleConfigOptionsRuntime    `mapstructure:"runtime" json:"runtime"`
}

type SimpleConfigOptionsRuntime struct {
	GPURequest    string                 `mapstructure:"gpuRequest" json:"gpuRequest,omitempty"`
	ServersMemory string                 `mapstructure:"serversMemory" json:"serversMemory,omitempty"`
	AgentsMemory  string                 `mapstructure:"agentsMemory" json:"agentsMemory,omitempty"`
	HostPidMode   bool                   `mapstructure:"hostPidMode" yjson:"hostPidMode,omitempty"`
	Labels        []LabelWithNodeFilters `mapstructure:"labels" json:"labels,omitempty"`
	Ulimits       []Ulimit               `mapstructure:"ulimits" json:"ulimits,omitempty"`
}

type Ulimit struct {
	Name string `mapstructure:"name" json:"name"`
	Soft int64  `mapstructure:"soft" json:"soft"`
	Hard int64  `mapstructure:"hard" json:"hard"`
}

type SimpleConfigOptionsK3d struct {
	Wait                bool                               `mapstructure:"wait" json:"wait"`
	Timeout             time.Duration                      `mapstructure:"timeout" json:"timeout,omitempty"`
	DisableLoadbalancer bool                               `mapstructure:"disableLoadbalancer" json:"disableLoadbalancer"`
	DisableImageVolume  bool                               `mapstructure:"disableImageVolume" json:"disableImageVolume"`
	NoRollback          bool                               `mapstructure:"disableRollback" json:"disableRollback"`
	NodeHookActions     []k3d.NodeHookAction               `mapstructure:"nodeHookActions" json:"nodeHookActions,omitempty"`
	Loadbalancer        SimpleConfigOptionsK3dLoadbalancer `mapstructure:"loadbalancer" json:"loadbalancer,omitempty"`
}

type SimpleConfigOptionsK3dLoadbalancer struct {
	ConfigOverrides []string `mapstructure:"configOverrides" json:"configOverrides,omitempty"`
}

type SimpleConfigOptionsK3s struct {
	ExtraArgs  []K3sArgWithNodeFilters `mapstructure:"extraArgs" json:"extraArgs,omitempty"`
	NodeLabels []LabelWithNodeFilters  `mapstructure:"nodeLabels" json:"nodeLabels,omitempty"`
}

type SimpleConfigRegistries struct {
	Use    []string                          `mapstructure:"use" json:"use,omitempty"`
	Create *SimpleConfigRegistryCreateConfig `mapstructure:"create" json:"create,omitempty"`
	Config string                            `mapstructure:"config" json:"config,omitempty"` // registries.yaml (k3s config for containerd registry override)
}

// SimpleConfig describes the toplevel k3d configuration file.
type SimpleConfig struct {
	config.TypeMeta   `mapstructure:",squash"`
	config.ObjectMeta `mapstructure:"metadata" json:"metadata,omitempty"`
	Servers           int                     `mapstructure:"servers" json:"servers,omitempty"` //nolint:lll    // default 1
	Agents            int                     `mapstructure:"agents" json:"agents,omitempty"`   //nolint:lll    // default 0
	ExposeAPI         SimpleExposureOpts      `mapstructure:"kubeAPI" json:"kubeAPI,omitempty"`
	Image             string                  `mapstructure:"image" json:"image,omitempty"`
	Network           string                  `mapstructure:"network" json:"network,omitempty"`
	Subnet            string                  `mapstructure:"subnet" json:"subnet,omitempty"`
	ClusterToken      string                  `mapstructure:"token" json:"clusterToken,omitempty"` // default: auto-generated
	Volumes           []VolumeWithNodeFilters `mapstructure:"volumes" json:"volumes,omitempty"`
	Ports             []PortWithNodeFilters   `mapstructure:"ports" json:"ports,omitempty"`
	Options           SimpleConfigOptions     `mapstructure:"options" json:"options,omitempty"`
	Env               []EnvVarWithNodeFilters `mapstructure:"env" json:"env,omitempty"`
	Registries        SimpleConfigRegistries  `mapstructure:"registries" json:"registries,omitempty"`
	HostAliases       []k3d.HostAlias         `mapstructure:"hostAliases" json:"hostAliases,omitempty"`
	Files             []FileWithNodeFilters   `mapstructure:"files" json:"files,omitempty"`
}

// SimpleExposureOpts provides a simplified syntax compared to the original k3d.ExposureOpts
type SimpleExposureOpts struct {
	Host     string `mapstructure:"host" json:"host,omitempty"`
	HostIP   string `mapstructure:"hostIP" json:"hostIP,omitempty"`
	HostPort string `mapstructure:"hostPort" json:"hostPort,omitempty"`
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
	config.TypeMeta   `mapstructure:",squash"`
	k3d.Cluster       `mapstructure:",squash"`
	ClusterCreateOpts k3d.ClusterCreateOpts         `mapstructure:"options" json:"options"`
	KubeconfigOpts    SimpleConfigOptionsKubeconfig `mapstructure:"kubeconfig" json:"kubeconfig"`
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
	config.TypeMeta `mapstructure:",squash"`
	Clusters        []k3d.Cluster `mapstructure:"clusters" json:"clusters"`
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
