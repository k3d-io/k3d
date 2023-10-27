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
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"

	"github.com/k3d-io/k3d/v5/pkg/config"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
)

// NewCmdConfigMigrate returns a new cobra command
func NewCmdConfigMigrate() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "migrate INPUT [OUTPUT]",
		Aliases: []string{"update"},
		Args:    cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			configFile := args[0]

			if _, err := os.Stat(configFile); err != nil {
				l.Log().Fatalf("Failed to stat config file %s: %+v", configFile, err)
			}

			cfgViper := viper.New()
			cfgViper.SetConfigType("yaml")

			cfgViper.SetConfigFile(configFile)

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

			if err := config.ValidateSchemaFile(configFile, schema); err != nil {
				l.Log().Fatalf("Schema Validation failed for config file %s: %+v", configFile, err)
			}

			l.Log().Infof("Using config file %s (%s#%s)", cfgViper.ConfigFileUsed(), strings.ToLower(cfgViper.GetString("apiVersion")), strings.ToLower(cfgViper.GetString("kind")))

			cfg, err := config.FromViper(cfgViper)
			if err != nil {
				l.Log().Fatalln(err)
			}

			if cfg.GetAPIVersion() != config.DefaultConfigApiVersion {
				cfg, err = config.Migrate(cfg, config.DefaultConfigApiVersion)
				if err != nil {
					l.Log().Fatalln(err)
				}
			}

			yamlout, err := yaml.Marshal(cfg)
			if err != nil {
				l.Log().Fatalln(err)
			}

			output := "-"

			if len(args) > 1 {
				output = args[1]
			}

			if output == "-" {
				if _, err := os.Stdout.Write(yamlout); err != nil {
					l.Log().Fatalln(err)
				}
			} else {
				if err := os.WriteFile(output, yamlout, os.ModePerm); err != nil {
					l.Log().Fatalln(err)
				}
			}
		},
	}

	return cmd
}
