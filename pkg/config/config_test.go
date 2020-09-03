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

	"github.com/go-test/deep"
	k3d "github.com/rancher/k3d/v3/pkg/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestReadSimpleConfig(t *testing.T) {

	expectedConfig := SimpleConfig{
		Kind:    "simple",
		Servers: 1,
		Agents:  2,
		ExposeAPI: k3d.ExposeAPI{
			HostIP: "0.0.0.0",
			Port:   "6443",
		},
		Image: "rancher/k3s:latest",
		Volumes: []string{
			"/my/path:/some/path",
		},
		Ports: []string{
			"80:80@loadbalancer",
			"0.0.0.0:443:443@loadbalancer",
		},
		Options: k3d.ClusterCreateOpts{
			WaitForServer: true,
		},
		KubeconfigOpts: ClusterCreateKubeconfigOptions{
			UpdateDefaultKubeconfig: true,
			SwitchCurrentContext:    true,
		},
	}

	CfgFile = "./config_test_simple.yaml"

	InitConfig()

	t.Logf("\n========== Read Config ==========\n%+v\n=================================\n%+v\n=================================\n", FileConfig, viper.AllSettings())

	if diff := deep.Equal(FileConfig, &expectedConfig); diff != nil {
		t.Errorf("Actual representation\n%+v\ndoes not match expected representation\n%+v\nDiff:\n%+v", FileConfig, expectedConfig, diff)
	}

}

func TestReadClusterConfig(t *testing.T) {

	expectedConfig := ClusterConfig{
		Kind: "cluster",
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

	CfgFile = "./config_test_cluster.yaml"

	InitConfig()

	t.Logf("\n========== Read Config ==========\n%+v\n=================================\n%+v\n=================================\n", FileConfig, viper.AllSettings())

	if diff := deep.Equal(FileConfig, &expectedConfig); diff != nil {
		t.Errorf("Actual representation\n%+v\ndoes not match expected representation\n%+v\nDiff:\n%+v", FileConfig, expectedConfig, diff)
	}

}

func TestReadClusterListConfig(t *testing.T) {

	expectedConfig := ClusterListConfig{
		Kind: "ClusterList",
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

	CfgFile = "./config_test_cluster_list.yaml"

	InitConfig()

	t.Logf("\n========== Read Config ==========\n%+v\n=================================\n%+v\n=================================\n", FileConfig, viper.AllSettings())

	if diff := deep.Equal(FileConfig, &expectedConfig); diff != nil {
		t.Errorf("Actual representation\n%+v\ndoes not match expected representation\n%+v\nDiff:\n%+v", FileConfig, expectedConfig, diff)
	}

}

func TestReadUnknownConfig(t *testing.T) {

	CfgFile = "./config_test_unknown.yaml"

	// catch fatal exit
	defer func() { log.StandardLogger().ExitFunc = nil }()
	var fatal bool
	log.StandardLogger().ExitFunc = func(int) { fatal = true }

	InitConfig()

	assert.Equal(t, true, fatal)

}
