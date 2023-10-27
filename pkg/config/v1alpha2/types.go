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

package v1alpha2

import (
	_ "embed"
	"fmt"
	"time"

	configtypes "github.com/k3d-io/k3d/v5/pkg/config/types"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/version"
)

// JSONSchema describes the schema used to validate config files
//
//go:embed schema.json
var JSONSchema string

const ApiVersion = "k3d.io/v1alpha2"

// DefaultConfigTpl for printing
const DefaultConfigTpl = `---
apiVersion: %s
kind: Simple
name: %s
servers: 1
agents: 0
image: %s
`

// DefaultConfig templated DefaultConfigTpl
var DefaultConfig = fmt.Sprintf(
	DefaultConfigTpl,
	ApiVersion,
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
	GPURequest    string `mapstructure:"gpuRequest,omitempty" json:"gpuRequest,omitempty"`
	ServersMemory string `mapstructure:"serversMemory,omitempty" json:"serversMemory,omitempty"`
	AgentsMemory  string `mapstructure:"agentsMemory,omitempty" json:"agentsMemory,omitempty"`
}

type SimpleConfigOptionsK3d struct {
	Wait                       bool                 `mapstructure:"wait" json:"wait"`
	Timeout                    time.Duration        `mapstructure:"timeout" json:"timeout,omitempty"`
	DisableLoadbalancer        bool                 `mapstructure:"disableLoadbalancer" json:"disableLoadbalancer"`
	DisableImageVolume         bool                 `mapstructure:"disableImageVolume" json:"disableImageVolume"`
	NoRollback                 bool                 `mapstructure:"disableRollback" json:"disableRollback"`
	PrepDisableHostIPInjection bool                 `mapstructure:"disableHostIPInjection" json:"disableHostIPInjection"`
	NodeHookActions            []k3d.NodeHookAction `mapstructure:"nodeHookActions" json:"nodeHookActions,omitempty"`
}

type SimpleConfigOptionsK3s struct {
	ExtraServerArgs []string `mapstructure:"extraServerArgs,omitempty" json:"extraServerArgs,omitempty"`
	ExtraAgentArgs  []string `mapstructure:"extraAgentArgs,omitempty" json:"extraAgentArgs,omitempty"`
}

// SimpleConfig describes the toplevel k3d configuration file.
type SimpleConfig struct {
	configtypes.TypeMeta `mapstructure:",squash"`
	Name                 string                  `mapstructure:"name" json:"name,omitempty"`
	Servers              int                     `mapstructure:"servers" json:"servers,omitempty"` //nolint:lll    // default 1
	Agents               int                     `mapstructure:"agents" json:"agents,omitempty"`   //nolint:lll    // default 0
	ExposeAPI            SimpleExposureOpts      `mapstructure:"kubeAPI" json:"kubeAPI,omitempty"`
	Image                string                  `mapstructure:"image" json:"image,omitempty"`
	Network              string                  `mapstructure:"network" json:"network,omitempty"`
	Subnet               string                  `mapstructure:"subnet" json:"subnet,omitempty"`
	ClusterToken         string                  `mapstructure:"token" json:"clusterToken,omitempty"` // default: auto-generated
	Volumes              []VolumeWithNodeFilters `mapstructure:"volumes" json:"volumes,omitempty"`
	Ports                []PortWithNodeFilters   `mapstructure:"ports" json:"ports,omitempty"`
	Labels               []LabelWithNodeFilters  `mapstructure:"labels" json:"labels,omitempty"`
	Options              SimpleConfigOptions     `mapstructure:"options" json:"options,omitempty"`
	Env                  []EnvVarWithNodeFilters `mapstructure:"env" json:"env,omitempty"`
	Registries           struct {
		Use    []string `mapstructure:"use" json:"use,omitempty"`
		Create bool     `mapstructure:"create" json:"create,omitempty"`
		Config string   `mapstructure:"config" json:"config,omitempty"` // registries.yaml (k3s config for containerd registry override)
	} `mapstructure:"registries" json:"registries,omitempty"`
}

// SimpleExposureOpts provides a simplified syntax compared to the original k3d.ExposureOpts
type SimpleExposureOpts struct {
	Host     string `mapstructure:"host" json:"host,omitempty"`
	HostIP   string `mapstructure:"hostIP" json:"hostIP,omitempty"`
	HostPort string `mapstructure:"hostPort" json:"hostPort,omitempty"`
}

// Kind implements Config.Kind
func (c SimpleConfig) GetKind() string {
	return "Simple"
}

func (c SimpleConfig) GetAPIVersion() string {
	return ApiVersion
}

// ClusterConfig describes a single cluster config
type ClusterConfig struct {
	configtypes.TypeMeta `mapstructure:",squash"`
	k3d.Cluster          `mapstructure:",squash"`
	ClusterCreateOpts    k3d.ClusterCreateOpts         `mapstructure:"options" json:"options"`
	KubeconfigOpts       SimpleConfigOptionsKubeconfig `mapstructure:"kubeconfig" json:"kubeconfig"`
}

// Kind implements Config.Kind
func (c ClusterConfig) GetKind() string {
	return "Simple"
}

func (c ClusterConfig) GetAPIVersion() string {
	return ApiVersion
}

// ClusterListConfig describes a list of clusters
type ClusterListConfig struct {
	configtypes.TypeMeta `mapstructure:",squash"`
	Clusters             []k3d.Cluster `mapstructure:"clusters" json:"clusters"`
}

func (c ClusterListConfig) GetKind() string {
	return "Simple"
}

func (c ClusterListConfig) GetAPIVersion() string {
	return ApiVersion
}

func GetConfigByKind(kind string) (configtypes.Config, error) {
	// determine config kind
	switch kind {
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
