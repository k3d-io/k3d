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
	"testing"

	configtypes "github.com/k3d-io/k3d/v5/pkg/config/types"
	conf "github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

func TestMergeSimpleConfig(t *testing.T) {
	srcConfig := "./test_assets/config_test_simple.yaml"
	destConfig := "./test_assets/config_test_simple_2.yaml"

	var src, dest configtypes.Config
	var err error

	cfg1 := viper.New()
	cfg1.SetConfigFile(srcConfig)
	_ = cfg1.ReadInConfig()

	cfg2 := viper.New()
	cfg2.SetConfigFile(destConfig)
	_ = cfg2.ReadInConfig()

	if src, err = FromViper(cfg1); err != nil {
		t.Fatal(err)
	}

	if dest, err = FromViper(cfg2); err != nil {
		t.Fatal(err)
	}

	mergedConfig, err := MergeSimple(dest.(conf.SimpleConfig), src.(conf.SimpleConfig))
	if err != nil {
		t.Fatal(err)
	}

	// ensure that we get the two filled fields of destConfig
	assert.Equal(t, mergedConfig.Name, dest.(conf.SimpleConfig).Name)
	assert.Equal(t, mergedConfig.Agents, dest.(conf.SimpleConfig).Agents)

	// ensure that we get the other fields from the srcConfig (only checking two of them here)
	assert.Equal(t, mergedConfig.Servers, src.(conf.SimpleConfig).Servers)
	assert.Equal(t, mergedConfig.Image, src.(conf.SimpleConfig).Image)
}
