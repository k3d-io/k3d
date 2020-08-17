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

	log "github.com/sirupsen/logrus"

	"github.com/mitchellh/go-homedir"
	k3d "github.com/rancher/k3d/v3/pkg/types"
	"github.com/spf13/viper"
)

// Config describes the toplevel k3d configuration file
type Config struct {
	Cluster           *k3d.Cluster           `yaml:"cluster" json:"cluster"`
	ClusterCreateOpts *k3d.ClusterCreateOpts `yaml:"clusterCreateOpts" json:"clusterCreateOpts"`
}

// CurrentConfig represents the currently active config
var CurrentConfig *Config

// CfgFile is the globally set config file
var CfgFile string

// InitConfig initializes viper
func InitConfig() {

	CurrentConfig := &Config{}

	viper.SetEnvPrefix(k3d.DefaultObjectNamePrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	viper.SetConfigType("yaml")

	if CfgFile != "" {
		viper.SetConfigFile(CfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			log.Fatalln(err)
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".k3d")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Debugln("No config file found!")
		} else { // config file found but some other error happened
			log.Debugf("Not using config file: %+v", viper.ConfigFileUsed())
			log.Debugln(err)
		}
	} else {

		if err := viper.Unmarshal(&CurrentConfig); err != nil {
			log.Warnln("Failed to unmarshal config")
			log.Warnln(err)
		}

		log.Infof("Using Config: %s", viper.ConfigFileUsed())
	}
}
