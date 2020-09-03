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
package config

import (
	"strings"

	"github.com/imdario/mergo"
	log "github.com/sirupsen/logrus"

	"github.com/mitchellh/go-homedir"
	k3d "github.com/rancher/k3d/v3/pkg/types"
	"github.com/spf13/viper"
)

// Config interface.
type Config interface {
	GetKind() string
}

// ClusterCreateKubeconfigOptions describes the set of options referring to the kubeconfig during cluster creation.
type ClusterCreateKubeconfigOptions struct {
	UpdateDefaultKubeconfig bool `mapstructure:"updateDefaultKubeconfig" yaml:"updateDefaultKubeconfig" json:"updateDefaultKubeconfig,omitempty"` // default: true
	SwitchCurrentContext    bool `mapstructure:"switchCurrentContext" yaml:"switchCurrentContext" json:"switchCurrentContext,omitempty"`          //nolint:lll    // default: true
}

// SimpleConfig describes the toplevel k3d configuration file.
type SimpleConfig struct {
	Kind                string                         `mapstructure:"kind" yaml:"kind" json:"kind,omitempty"`
	Name                string                         `mapstructure:"name" yaml:"name" json:"name,omitempty"`
	Servers             int                            `mapstructure:"servers" yaml:"servers" json:"servers,omitempty"` //nolint:lll    // default 1
	Agents              int                            `mapstructure:"agents" yaml:"agents" json:"agents,omitempty"`    //nolint:lll    // default 0
	ExposeAPI           k3d.ExposeAPI                  `mapstructure:"exposeAPI" yaml:"exposeAPI" json:"exposeAPI,omitempty"`
	Image               string                         `mapstructure:"image" yaml:"image" json:"image,omitempty"`
	Network             string                         `mapstructure:"network" yaml:"network" json:"network,omitempty"`
	ClusterToken        string                         `mapstructure:"clusterToken" yaml:"clusterToken" json:"clusterToken,omitempty"` // default: auto-generated
	Volumes             []string                       `mapstructure:"volumes" yaml:"volumes" json:"volumes,omitempty"`
	Ports               []string                       `mapstructure:"ports" yaml:"ports" json:"ports,omitempty"`
	Options             k3d.ClusterCreateOpts          `mapstructure:"options" yaml:"options" json:"options,omitempty"`
	KubeconfigOpts      ClusterCreateKubeconfigOptions `mapstructure:"kubeconfig" yaml:"kubeconfig" json:"kubeconfig,omitempty"`
	LoadBalancerEnabled bool                           `mapstructure:"loadbalancerEnabled" yaml:"loadbalancerEnabled" json:"loadbalancerEnabled,omitempty"`
}

// GetKind implements Config.GetKind
func (c SimpleConfig) GetKind() string {
	return c.Kind
}

// ClusterConfig describes a single cluster config
type ClusterConfig struct {
	Kind    string      `mapstructure:"kind" yaml:"kind" json:"kind,omitempty"`
	Cluster k3d.Cluster `mapstructure:",squash" yaml:",inline"`
}

// GetKind implements Config.GetKind
func (c ClusterConfig) GetKind() string {
	return c.Kind
}

// ClusterListConfig describes a list of clusters
type ClusterListConfig struct {
	Kind     string        `mapstructure:"kind" yaml:"kind" json:"kind,omitempty"`
	Clusters []k3d.Cluster `mapstructure:"clusters" yaml:"clusters"`
}

// GetKind implements Config.GetKind
func (c ClusterListConfig) GetKind() string {
	return c.Kind
}

// FileConfig represents the currently active config
var FileConfig Config

var CLIConfig SimpleConfig

var FileViper *viper.Viper

var CLIViper *viper.Viper

// CfgFile is the globally set config file
var CfgFile string

// InitConfig initializes viper
func InitConfig() {

	CLIViper = viper.GetViper()

	FileViper = viper.New()
	FileViper.SetConfigType("yaml")

	log.Debugf("Config at the start: %+v", viper.AllSettings())

	CLIViper.SetEnvPrefix(k3d.DefaultObjectNamePrefix)
	CLIViper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	CLIViper.AutomaticEnv()

	// lookup config file
	if CfgFile != "" {
		FileViper.SetConfigFile(CfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			log.Fatalln(err)
		}
		FileViper.AddConfigPath(home)
		FileViper.SetConfigName(".k3d")
	}

	// try to read config into memory (viper map structure)
	if err := FileViper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Debugln("No config file found!")
		} else { // config file found but some other error happened
			log.Debugf("Not using config file: %+v", FileViper.ConfigFileUsed())
			log.Debugln(err)
		}
		return
	}

	// determine config kind
	switch strings.ToLower(FileViper.GetString("kind")) {
	case "simple":
		FileConfig = &SimpleConfig{}
	case "cluster":
		FileConfig = &ClusterConfig{}
	case "clusterlist":
		FileConfig = &ClusterListConfig{}
	case "":
		log.Fatalln("Missing `kind` in config file")
	default:
		log.Fatalf("Unknown `kind` '%s' in config file", FileViper.GetString("kind"))
	}

	// parse config to struct
	if err := CLIViper.Unmarshal(&CLIConfig); err != nil {
		log.Warnln("Failed to unmarshal CLI config")
		log.Warnln(err)
	}

	if err := FileViper.Unmarshal(&FileConfig); err != nil {
		log.Warnln("Failed to unmarshal File config")
		log.Warnln(err)
	}
	log.Infof("Using Config: %s", FileViper.ConfigFileUsed())

}

func GetMergedConfig() (Config, error) {
	mergedConfig := FileConfig

	if err := mergo.MergeWithOverwrite(mergedConfig, CLIConfig, nil); err != nil {
		log.Errorln("Failed to merge configs")
		return nil, err
	}

	log.Debugf("MERGED: %+v", mergedConfig)

	return mergedConfig, nil

}
