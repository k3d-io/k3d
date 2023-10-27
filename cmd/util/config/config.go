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
package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"

	"github.com/k3d-io/k3d/v5/cmd/util"
	"github.com/k3d-io/k3d/v5/pkg/config"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
)

func InitViperWithConfigFile(cfgViper *viper.Viper, configFile string) error {
	// viper for the general config (file, env and non pre-processed flags)
	cfgViper.SetEnvPrefix("K3D")
	cfgViper.AutomaticEnv()
	cfgViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	cfgViper.SetConfigType("yaml")

	// Set config file, if specified
	if configFile != "" {
		streams := util.StandardIOStreams()
		//flag to mark from where we read the config
		fromStdIn := false
		if configFile == "-" {
			fromStdIn = true
		} else if _, err := os.Stat(configFile); err != nil {
			l.Log().Fatalf("Failed to stat config file %s: %+v", configFile, err)
		}

		// create temporary file to expand environment variables in the config without writing that back to the original file
		// we're doing it here, because this happens just before absolutely all other processing
		tmpfile, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("k3d-config-tmp-%s", filepath.Base(configFile)))
		if err != nil {
			l.Log().Fatalf("error creating temp copy of configfile %s for variable expansion: %v", configFile, err)
		}
		defer tmpfile.Close()

		var originalcontent []byte
		if fromStdIn {
			// otherwise read from stdin
			originalcontent, err = io.ReadAll(streams.In)
			if err != nil {
				l.Log().Fatalf("Failed to read config file from stdin: %+v", err)
			}
		} else {
			originalcontent, err = os.ReadFile(configFile)
			if err != nil {
				l.Log().Fatalf("error reading config file %s: %v", configFile, err)
			}
		}

		expandedcontent := os.ExpandEnv(string(originalcontent))
		if _, err := tmpfile.WriteString(expandedcontent); err != nil {
			l.Log().Fatalf("error writing expanded config file contents to temp file %s: %v", tmpfile.Name(), err)
		}

		// use temp file with expanded variables
		cfgViper.SetConfigFile(tmpfile.Name())

		// try to read config into memory (viper map structure)
		if err := cfgViper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				l.Log().Fatalf("Config file %s not found: %+v", configFile, err)
			}
			// config file found but some other error happened
			l.Log().Fatalf("Failed to read config file %s: %+v", configFile, err)
		}

		schema, err := config.GetSchemaByVersion(cfgViper.GetString("apiVersion"))
		if err != nil {
			l.Log().Fatalf("Cannot validate config file %s: %+v", configFile, err)
		}

		if err := config.ValidateSchemaFile(tmpfile.Name(), schema); err != nil {
			l.Log().Fatalf("Schema Validation failed for config file %s: %+v", configFile, err)
		}

		l.Log().Infof("Using config file %s (%s#%s)", configFile, strings.ToLower(cfgViper.GetString("apiVersion")), strings.ToLower(cfgViper.GetString("kind")))
	}
	if l.Log().GetLevel() >= logrus.DebugLevel {
		c, _ := yaml.Marshal(cfgViper.AllSettings())
		l.Log().Debugf("Configuration:\n%s", c)
	}
	return nil
}
