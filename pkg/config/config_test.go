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
	"github.com/spf13/viper"
)

func TestReadConfig(t *testing.T) {

	expectedConfig := Config{
		Cluster: k3d.Cluster{
			Name: "test-cluster",
			Nodes: []*k3d.Node{
				&k3d.Node{
					Name: "test-node-0",
					Role: k3d.ServerRole,
				},
				&k3d.Node{
					Name: "test-node-1",
					Role: k3d.AgentRole,
				},
			},
		},
	}

	viper.SetConfigFile("./config_test.yaml")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		t.Error(err)
	}

	var readConfig *Config

	if err := viper.Unmarshal(&readConfig); err != nil {
		t.Error(err)
	}

	t.Logf("\n========== Read Config ==========\n%+v\n=================================\n", readConfig)

	if diff := deep.Equal(*readConfig, expectedConfig); diff != nil {
		t.Errorf("Actual representation\n%+v\ndoes not match expected representation\n%+v\nDiff:\n%+v", readConfig, expectedConfig, diff)
	}

}
