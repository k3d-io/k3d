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
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"

	conf "github.com/rancher/k3d/v4/pkg/config/v1alpha1"
	k3d "github.com/rancher/k3d/v4/pkg/types"
)

func ReadConfig(file string) (conf.Config, error) {
	cfgViper := viper.New()

	cfgViper.SetConfigFile(file)

	cfgViper.SetConfigType("yaml")
	cfgViper.SetEnvPrefix(k3d.DefaultObjectNamePrefix)
	cfgViper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	cfgViper.AutomaticEnv()

	// try to read config into memory (viper map structure)
	if err := cfgViper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Errorln("No config file found!")

			return nil, err
		}
		// config file found but some other error happened
		log.Debugf("Failed to read config file: %+v", cfgViper.ConfigFileUsed())
		return nil, err
	}

	var cfg conf.Config

	// determine config kind
	switch strings.ToLower(cfgViper.GetString("kind")) {
	case "simple":
		cfg = conf.SimpleConfig{}
	case "cluster":
		cfg = conf.ClusterConfig{}
	case "clusterlist":
		cfg = conf.ClusterListConfig{}
	case "":
		return nil, fmt.Errorf("Missing `kind` in config file")
	default:
		return nil, fmt.Errorf("Unknown `kind` '%s' in config file", cfgViper.GetString("kind"))
	}

	if err := cfgViper.Unmarshal(&cfg); err != nil {
		log.Errorln("Failed to unmarshal File config")

		return nil, err
	}

	log.Infof("Using Config: %s", cfgViper.ConfigFileUsed())

	return cfg, nil
}
