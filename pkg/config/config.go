/*
Copyright © 2020-2021 The k3d Author(s)

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

	l "github.com/rancher/k3d/v5/pkg/logger"

	"github.com/spf13/viper"

	"github.com/rancher/k3d/v5/pkg/config/v1alpha2"
	"github.com/rancher/k3d/v5/pkg/config/v1alpha3"
	defaultConfig "github.com/rancher/k3d/v5/pkg/config/v1alpha3"

	types "github.com/rancher/k3d/v5/pkg/config/types"
)

const DefaultConfigApiVersion = defaultConfig.ApiVersion

var Schemas = map[string]string{
	v1alpha2.ApiVersion: v1alpha2.JSONSchema,
	v1alpha3.ApiVersion: v1alpha3.JSONSchema,
}

func GetSchemaByVersion(apiVersion string) ([]byte, error) {
	schema, ok := Schemas[strings.ToLower(apiVersion)]
	if !ok {
		return nil, fmt.Errorf("unsupported apiVersion '%s'", apiVersion)
	}
	return []byte(schema), nil
}

func FromViper(config *viper.Viper) (types.Config, error) {

	var cfg types.Config
	var err error

	apiVersion := strings.ToLower(config.GetString("apiversion"))
	kind := strings.ToLower(config.GetString("kind"))

	l.Log().Tracef("Trying to read config apiVersion='%s', kind='%s'", apiVersion, kind)

	switch apiVersion {
	case "k3d.io/v1alpha2":
		cfg, err = v1alpha2.GetConfigByKind(kind)
	case "k3d.io/v1alpha3":
		cfg, err = v1alpha3.GetConfigByKind(kind)
	case "":
		cfg, err = defaultConfig.GetConfigByKind(kind)
	default:
		return nil, fmt.Errorf("cannot read config with apiversion '%s'", config.GetString("apiversion"))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse config '%s': %w'", config.ConfigFileUsed(), err)
	}

	if err := config.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file '%s': %w", config.ConfigFileUsed(), err)
	}

	return cfg, nil
}

func getMigrations(version string) map[string]func(types.Config) (types.Config, error) {
	switch version {
	case v1alpha3.ApiVersion:
		return v1alpha3.Migrations
	default:
		return nil
	}
}
