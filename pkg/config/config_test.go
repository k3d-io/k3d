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
	"time"

	"github.com/go-test/deep"
	configtypes "github.com/k3d-io/k3d/v5/pkg/config/types"
	conf "github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	"github.com/spf13/viper"

	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

func TestReadSimpleConfig(t *testing.T) {
	exposedAPI := conf.SimpleExposureOpts{}
	exposedAPI.HostIP = "0.0.0.0"
	exposedAPI.HostPort = "6443"

	expectedConfig := conf.SimpleConfig{
		TypeMeta: configtypes.TypeMeta{
			APIVersion: "k3d.io/v1alpha5",
			Kind:       "Simple",
		},
		ObjectMeta: configtypes.ObjectMeta{
			Name: "test",
		},
		Servers:   1,
		Agents:    2,
		ExposeAPI: exposedAPI,
		Image:     "rancher/k3s:latest",
		Volumes: []conf.VolumeWithNodeFilters{
			{
				Volume:      "/my/path:/some/path",
				NodeFilters: []string{"all"},
			},
		},
		Ports: []conf.PortWithNodeFilters{
			{
				Port:        "80:80",
				NodeFilters: []string{"loadbalancer"},
			}, {
				Port:        "0.0.0.0:443:443",
				NodeFilters: []string{"loadbalancer"},
			},
		},
		Env: []conf.EnvVarWithNodeFilters{
			{
				EnvVar:      "bar=baz",
				NodeFilters: []string{"all"},
			},
		},
		Files: []conf.FileWithNodeFilters{
			{
				Description: "Source: Embedded, Destination: Absolute path",
				Source: "apiVersion: v1\n" +
					"kind: Namespace\n" +
					"metadata:\n" +
					"  name: foo\n",
				Destination: "/var/lib/rancher/k3s/server/manifests/foo.yaml",
			},
			{
				Description: "Source: Embedded, Destination: Magic shortcut path",
				Source: "apiVersion: v1\n" +
					"kind: Namespace\n" +
					"metadata:\n" +
					"  name: bar\n",
				Destination: "k3s-manifests/bar.yaml",
			},
			{
				Description: "Source: Relative, Destination: Magic shortcut path",
				Source:      "ns-baz.yaml",
				Destination: "k3s-manifests-custom/baz.yaml",
			},
			{
				Description: "Source: Relative, Destination: Magic shortcut path, Node: Servers only",
				Source:      "ns-baz.yaml",
				Destination: "k3s-manifests/baz.yaml",
				NodeFilters: []string{"server:*"},
			},
		},
		Options: conf.SimpleConfigOptions{
			K3dOptions: conf.SimpleConfigOptionsK3d{
				Wait:                true,
				Timeout:             60 * time.Second,
				DisableLoadbalancer: false,
				DisableImageVolume:  false,
			},
			K3sOptions: conf.SimpleConfigOptionsK3s{
				ExtraArgs: []conf.K3sArgWithNodeFilters{
					{
						Arg:         "--tls-san=127.0.0.1",
						NodeFilters: []string{"server:*"},
					},
				},
				NodeLabels: []conf.LabelWithNodeFilters{
					{
						Label:       "foo=bar",
						NodeFilters: []string{"server:0", "loadbalancer"},
					},
				},
			},
			KubeconfigOptions: conf.SimpleConfigOptionsKubeconfig{
				UpdateDefaultKubeconfig: true,
				SwitchCurrentContext:    true,
			},
			Runtime: conf.SimpleConfigOptionsRuntime{
				Labels: []conf.LabelWithNodeFilters{
					{
						Label:       "foo=bar",
						NodeFilters: []string{"server:0", "loadbalancer"},
					},
				},
				Ulimits: []conf.Ulimit{{
					Name: "nofile",
					Soft: 1024,
					Hard: 1024,
				}},
			},
		},
	}

	cfgFile := "./test_assets/config_test_simple.yaml"

	config := viper.New()
	config.SetConfigFile(cfgFile)

	// try to read config into memory (viper map structure)
	if err := config.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			t.Error(err)
		}
		// config file found but some other error happened
		t.Error(err)
	}

	cfg, err := FromViper(config)
	if err != nil {
		t.Error(err)
	}

	t.Logf("\n========== Read Config %s ==========\n%+v\n=================================\n", config.ConfigFileUsed(), cfg)

	if diff := deep.Equal(cfg, expectedConfig); diff != nil {
		t.Errorf("Actual representation\n%+v\ndoes not match expected representation\n%+v\nDiff:\n%+v", cfg, expectedConfig, diff)
	}
}

func TestReadClusterConfig(t *testing.T) {
	expectedConfig := conf.ClusterConfig{
		TypeMeta: configtypes.TypeMeta{
			APIVersion: "k3d.io/v1alpha5",
			Kind:       "Cluster",
		},
		Cluster: k3d.Cluster{
			Name: "foo",
			Nodes: []*k3d.Node{
				{
					Name: "foo-node-0",
					Role: k3d.ServerRole,
				},
			},
		},
	}

	cfgFile := "./test_assets/config_test_cluster.yaml"

	config := viper.New()
	config.SetConfigFile(cfgFile)

	// try to read config into memory (viper map structure)
	if err := config.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			t.Error(err)
		}
		// config file found but some other error happened
		t.Error(err)
	}

	readConfig, err := FromViper(config)
	if err != nil {
		t.Error(err)
	}

	t.Logf("\n========== Read Config ==========\n%+v\n=================================\n", readConfig)

	if diff := deep.Equal(readConfig, expectedConfig); diff != nil {
		t.Errorf("Actual representation\n%+v\ndoes not match expected representation\n%+v\nDiff:\n%+v", readConfig, expectedConfig, diff)
	}
}

func TestReadClusterListConfig(t *testing.T) {
	expectedConfig := conf.ClusterListConfig{
		TypeMeta: configtypes.TypeMeta{
			APIVersion: "k3d.io/v1alpha5",
			Kind:       "ClusterList",
		},
		Clusters: []k3d.Cluster{
			{
				Name: "foo",
				Nodes: []*k3d.Node{
					{
						Name: "foo-node-0",
						Role: k3d.ServerRole,
					},
				},
			},
			{
				Name: "bar",
				Nodes: []*k3d.Node{
					{
						Name: "bar-node-0",
						Role: k3d.ServerRole,
					},
				},
			},
		},
	}

	cfgFile := "./test_assets/config_test_cluster_list.yaml"

	config := viper.New()
	config.SetConfigFile(cfgFile)

	// try to read config into memory (viper map structure)
	if err := config.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			t.Error(err)
		}
		// config file found but some other error happened
		t.Error(err)
	}

	readConfig, err := FromViper(config)
	if err != nil {
		t.Error(err)
	}

	t.Logf("\n========== Read Config ==========\n%+v\n=================================\n", readConfig)

	if diff := deep.Equal(readConfig, expectedConfig); diff != nil {
		t.Errorf("Actual representation\n%+v\ndoes not match expected representation\n%+v\nDiff:\n%+v", readConfig, expectedConfig, diff)
	}
}

func TestReadUnknownConfig(t *testing.T) {
	cfgFile := "./test_assets/config_test_unknown.yaml"

	config := viper.New()
	config.SetConfigFile(cfgFile)

	// try to read config into memory (viper map structure)
	if err := config.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			t.Error(err)
		}
		// config file found but some other error happened
		t.Error(err)
	}

	_, err := FromViper(config)
	if err == nil {
		t.Fail()
	}
}

func TestReadSimpleConfigRegistries(t *testing.T) {
	exposedAPI := conf.SimpleExposureOpts{}
	exposedAPI.HostIP = "0.0.0.0"
	exposedAPI.HostPort = "6443"

	expectedConfig := conf.SimpleConfig{
		TypeMeta: configtypes.TypeMeta{
			APIVersion: "k3d.io/v1alpha5",
			Kind:       "Simple",
		},
		ObjectMeta: configtypes.ObjectMeta{
			Name: "test",
		},
		Servers: 1,
		Agents:  1,
		Registries: conf.SimpleConfigRegistries{
			Create: &conf.SimpleConfigRegistryCreateConfig{
				Name:     "registry.localhost",
				Host:     "0.0.0.0",
				HostPort: "5001",
			},
		},
	}

	cfgFile := "./test_assets/config_test_registries.yaml"

	config := viper.New()
	config.SetConfigFile(cfgFile)

	// try to read config into memory (viper map structure)
	if err := config.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			t.Error(err)
		}
		// config file found but some other error happened
		t.Error(err)
	}

	readConfig, err := FromViper(config)
	if err != nil {
		t.Error(err)
	}

	t.Logf("\n========== Read Config ==========\n%+v\n=================================\n", readConfig)

	if diff := deep.Equal(readConfig, expectedConfig); diff != nil {
		t.Errorf("Actual representation\n%+v\ndoes not match expected representation\n%+v\nDiff:\n%+v", readConfig, expectedConfig, diff)
	}
}
