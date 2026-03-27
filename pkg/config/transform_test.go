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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	"github.com/spf13/viper"
)

func TestTransformSimpleConfigToClusterConfig(t *testing.T) {
	cfgFile := "./test_assets/config_test_simple.yaml"

	vip := viper.New()
	vip.SetConfigFile(cfgFile)
	_ = vip.ReadInConfig()

	cfg, err := FromViper(vip)
	if err != nil {
		t.Error(err)
	}

	t.Logf("\n========== Read Config ==========\n%+v\n=================================\n", cfg)

	clusterCfg, err := TransformSimpleToClusterConfig(context.Background(), runtimes.Docker, cfg.(conf.SimpleConfig), cfgFile)
	if err != nil {
		t.Error(err)
	}

	t.Logf("\n===== Resulting Cluster Config =====\n%+v\n===============\n", clusterCfg)
}

func TestTransformEmptyRegistryCreateIsNil(t *testing.T) {
	// Simulate a config that has registries.use but no registries.create.
	// Viper unmarshaling can produce a non-nil but zero-value Create pointer
	// when the registries namespace has data from registries.use.
	simpleCfg := conf.SimpleConfig{
		Servers: 1,
		Registries: conf.SimpleConfigRegistries{
			Use:    []string{"k3d-myregistry"},
			Create: &conf.SimpleConfigRegistryCreateConfig{}, // empty, non-nil
		},
	}
	simpleCfg.Name = "emptyregtest"

	clusterCfg, err := TransformSimpleToClusterConfig(context.Background(), runtimes.Docker, simpleCfg, "")
	require.NoError(t, err)

	assert.Nil(t, clusterCfg.ClusterCreateOpts.Registries.Create,
		"empty Registries.Create should be treated as nil after transform")
	assert.Len(t, clusterCfg.ClusterCreateOpts.Registries.Use, 1,
		"Registries.Use should still have the referenced registry")
}

func TestTransformRegistryUseOnlyConfig(t *testing.T) {
	// Test loading a config file that only has registries.use (no create)
	cfgFile := "./test_assets/config_test_registry_use_only.yaml"

	vip := viper.New()
	vip.SetConfigFile(cfgFile)
	err := vip.ReadInConfig()
	require.NoError(t, err)

	cfg, err := FromViper(vip)
	require.NoError(t, err)

	simpleCfg := cfg.(conf.SimpleConfig)
	assert.Nil(t, simpleCfg.Registries.Create,
		"Registries.Create should be nil when only use is specified")
	assert.Len(t, simpleCfg.Registries.Use, 1)

	clusterCfg, err := TransformSimpleToClusterConfig(context.Background(), runtimes.Docker, simpleCfg, cfgFile)
	require.NoError(t, err)

	assert.Nil(t, clusterCfg.ClusterCreateOpts.Registries.Create,
		"Registries.Create should remain nil after transform")
	assert.Len(t, clusterCfg.ClusterCreateOpts.Registries.Use, 1,
		"Registries.Use should have one entry")
	assert.Equal(t, "k3d-registry-use-test-registry", clusterCfg.ClusterCreateOpts.Registries.Use[0].Host)
}
