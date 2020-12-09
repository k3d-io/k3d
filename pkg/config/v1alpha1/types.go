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

package v1alpha1

import (
	"fmt"
	"time"

	k3d "github.com/rancher/k3d/v4/pkg/types"
	"github.com/rancher/k3d/v4/version"
)

// DefaultConfigTpl for printing
const DefaultConfigTpl = `---
apiVersion: k3d.io/v1alpha1
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
	fmt.Sprintf("%s:%s", k3d.DefaultK3sImageRepo, version.GetK3sVersion(false)),
)

// TypeMeta, basically copied from https://github.com/kubernetes/apimachinery/blob/a3b564b22db316a41e94fdcffcf9995424fe924c/pkg/apis/meta/v1/types.go#L36-L56
type TypeMeta struct {
	Kind       string `mapstructure:"kind,omitempty" yaml:"kind,omitempty" json:"kind,omitempty"`
	APIVersion string `mapstructure:"apiVersion,omitempty" yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`
}

// Config interface.
type Config interface {
	GetKind() string
}

type VolumeWithNodeFilters struct {
	Volume      string   `mapstructure:"volume" yaml:"volume" json:"volume,omitempty"`
	NodeFilters []string `mapstructure:"nodeFilters" yaml:"nodeFilters" json:"nodeFilters,omitempty"`
}

type PortWithNodeFilters struct {
	Port        string   `mapstructure:"port" yaml:"port" json:"port,omitempty"`
	NodeFilters []string `mapstructure:"nodeFilters" yaml:"nodeFilters" json:"nodeFilters,omitempty"`
}

type LabelWithNodeFilters struct {
	Label       string   `mapstructure:"label" yaml:"label" json:"label,omitempty"`
	NodeFilters []string `mapstructure:"nodeFilters" yaml:"nodeFilters" json:"nodeFilters,omitempty"`
}

type EnvVarWithNodeFilters struct {
	EnvVar      string   `mapstructure:"envVar" yaml:"envVar" json:"envVar,omitempty"`
	NodeFilters []string `mapstructure:"nodeFilters" yaml:"nodeFilters" json:"nodeFilters,omitempty"`
}

// SimpleConfigOptionsKubeconfig describes the set of options referring to the kubeconfig during cluster creation.
type SimpleConfigOptionsKubeconfig struct {
	UpdateDefaultKubeconfig bool `mapstructure:"updateDefaultKubeconfig" yaml:"updateDefaultKubeconfig" json:"updateDefaultKubeconfig,omitempty"` // default: true
	SwitchCurrentContext    bool `mapstructure:"switchCurrentContext" yaml:"switchCurrentContext" json:"switchCurrentContext,omitempty"`          //nolint:lll    // default: true
}

type SimpleConfigOptions struct {
	K3dOptions        SimpleConfigOptionsK3d        `mapstructure:"k3d" yaml:"k3d"`
	K3sOptions        SimpleConfigOptionsK3s        `mapstructure:"k3s" yaml:"k3s"`
	KubeconfigOptions SimpleConfigOptionsKubeconfig `mapstructure:"kubeconfig" yaml:"kubeconfig"`
	Runtime           SimpleConfigOptionsRuntime    `mapstructure:"runtime" yaml:"runtime"`
}

type SimpleConfigOptionsRuntime struct {
	GPURequest string `mapstructure:"gpuRequest" yaml:"gpuRequest"`
}

type SimpleConfigOptionsK3d struct {
	Wait                       bool                 `mapstructure:"wait" yaml:"wait"`
	Timeout                    time.Duration        `mapstructure:"timeout" yaml:"timeout"`
	DisableLoadbalancer        bool                 `mapstructure:"disableLoadbalancer" yaml:"disableLoadbalancer"`
	DisableImageVolume         bool                 `mapstructure:"disableImageVolume" yaml:"disableImageVolume"`
	NoRollback                 bool                 `mapstructure:"noRollback" yaml:"noRollback"`
	PrepDisableHostIPInjection bool                 `mapstructure:"prepDisableHostIPInjection" yaml:"prepDisableHostIPInjection"`
	NodeHookActions            []k3d.NodeHookAction `mapstructure:"nodeHookActions" yaml:"nodeHookActions,omitempty"`
}

type SimpleConfigOptionsK3s struct {
	ExtraServerArgs []string `mapstructure:"extraServerArgs" yaml:"extraServerArgs"`
	ExtraAgentArgs  []string `mapstructure:"extraAgentArgs" yaml:"extraAgentArgs"`
}

// SimpleConfig describes the toplevel k3d configuration file.
type SimpleConfig struct {
	TypeMeta     `mapstructure:",squash" yaml:",inline"`
	Name         string                  `mapstructure:"name" yaml:"name" json:"name,omitempty"`
	Servers      int                     `mapstructure:"servers" yaml:"servers" json:"servers,omitempty"` //nolint:lll    // default 1
	Agents       int                     `mapstructure:"agents" yaml:"agents" json:"agents,omitempty"`    //nolint:lll    // default 0
	ExposeAPI    k3d.ExposePort          `mapstructure:"exposeAPI" yaml:"exposeAPI" json:"exposeAPI,omitempty"`
	Image        string                  `mapstructure:"image" yaml:"image" json:"image,omitempty"`
	Network      string                  `mapstructure:"network" yaml:"network" json:"network,omitempty"`
	ClusterToken string                  `mapstructure:"clusterToken" yaml:"clusterToken" json:"clusterToken,omitempty"` // default: auto-generated
	Volumes      []VolumeWithNodeFilters `mapstructure:"volumes" yaml:"volumes" json:"volumes,omitempty"`
	Ports        []PortWithNodeFilters   `mapstructure:"ports" yaml:"ports" json:"ports,omitempty"`
	Labels       []LabelWithNodeFilters  `mapstructure:"labels" yaml:"labels" json:"labels,omitempty"`
	Options      SimpleConfigOptions     `mapstructure:"options" yaml:"options" json:"options,omitempty"`
	Env          []EnvVarWithNodeFilters `mapstructure:"env" yaml:"env" json:"env,omitempty"`
	Registries   struct {
		Use    []*k3d.ExternalRegistry
		Create bool
	}
}

// GetKind implements Config.GetKind
func (c SimpleConfig) GetKind() string {
	return "Cluster"
}

// ClusterConfig describes a single cluster config
type ClusterConfig struct {
	TypeMeta          `mapstructure:",squash" yaml:",inline"`
	Cluster           k3d.Cluster                   `mapstructure:",squash" yaml:",inline"`
	ClusterCreateOpts k3d.ClusterCreateOpts         `mapstructure:"options" yaml:"options"`
	KubeconfigOpts    SimpleConfigOptionsKubeconfig `mapstructure:"kubeconfig" yaml:"kubeconfig"`
}

// GetKind implements Config.GetKind
func (c ClusterConfig) GetKind() string {
	return "Cluster"
}

// ClusterListConfig describes a list of clusters
type ClusterListConfig struct {
	TypeMeta `mapstructure:",squash" yaml:",inline"`
	Clusters []k3d.Cluster `mapstructure:"clusters" yaml:"clusters"`
}

func (c ClusterListConfig) GetKind() string {
	return "ClusterList"
}
