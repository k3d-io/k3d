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
	"testing"
	"time"

	"github.com/go-test/deep"
	conf "github.com/rancher/k3d/v4/pkg/config/v1alpha1"

	k3d "github.com/rancher/k3d/v4/pkg/types"
)

func TestReadSimpleConfig(t *testing.T) {

	expectedConfig := conf.SimpleConfig{
		TypeMeta: conf.TypeMeta{
			APIVersion: "k3d.io/v1alpha1",
			Kind:       "Simple",
		},
		Name:    "test",
		Servers: 1,
		Agents:  2,
		ExposeAPI: k3d.ExposePort{
			HostIP: "0.0.0.0",
			Port:   "6443",
		},
		Image: "rancher/k3s:latest",
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
		Labels: []conf.LabelWithNodeFilters{
			{
				Label:       "foo=bar",
				NodeFilters: []string{"server[0]", "loadbalancer"},
			},
		},
		Env: []conf.EnvVarWithNodeFilters{
			{
				EnvVar:      "bar=baz",
				NodeFilters: []string{"all"},
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
				ExtraServerArgs: []string{"--tls-san=127.0.0.1"},
				ExtraAgentArgs:  []string{},
			},
			KubeconfigOptions: conf.SimpleConfigOptionsKubeconfig{
				UpdateDefaultKubeconfig: true,
				SwitchCurrentContext:    true,
			},
		},
	}

	cfgFile := "./test_assets/config_test_simple.yaml"

	cfg, err := ReadConfig(cfgFile)
	if err != nil {
		t.Error(err)
	}

	simpleCfg, ok := cfg.(conf.SimpleConfig)
	if !ok {
		t.Error("Config is not of type SimpleConfig")
	}

	t.Logf("\n========== Read Config ==========\n%+v\n=================================\n", simpleCfg)

	if diff := deep.Equal(simpleCfg, expectedConfig); diff != nil {
		t.Errorf("Actual representation\n%+v\ndoes not match expected representation\n%+v\nDiff:\n%+v", simpleCfg, expectedConfig, diff)
	}

}

func TestReadClusterConfig(t *testing.T) {

	expectedConfig := conf.ClusterConfig{
		TypeMeta: conf.TypeMeta{
			APIVersion: "k3d.io/v1alpha1",
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

	readConfig, err := ReadConfig(cfgFile)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("\n========== Read Config ==========\n%+v\n=================================\n", readConfig)

	if diff := deep.Equal(readConfig, expectedConfig); diff != nil {
		t.Errorf("Actual representation\n%+v\ndoes not match expected representation\n%+v\nDiff:\n%+v", readConfig, expectedConfig, diff)
	}

}

func TestReadClusterListConfig(t *testing.T) {

	expectedConfig := conf.ClusterListConfig{
		TypeMeta: conf.TypeMeta{
			APIVersion: "k3d.io/v1alpha1",
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

	readConfig, err := ReadConfig(cfgFile)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("\n========== Read Config ==========\n%+v\n=================================\n", readConfig)

	if diff := deep.Equal(readConfig, expectedConfig); diff != nil {
		t.Errorf("Actual representation\n%+v\ndoes not match expected representation\n%+v\nDiff:\n%+v", readConfig, expectedConfig, diff)
	}

}

func TestReadUnknownConfig(t *testing.T) {

	cfgFile := "./test_assets/config_test_unknown.yaml"

	_, err := ReadConfig(cfgFile)
	if err == nil {
		t.Fail()
	}

}
