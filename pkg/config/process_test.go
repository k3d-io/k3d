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
	"context"
	"testing"

	"github.com/rancher/k3d/v4/pkg/runtimes"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

func TestProcessClusterConfig(t *testing.T) {
	cfgFile := "./test_assets/config_test_simple.yaml"

	vip := viper.New()
	vip.SetConfigFile(cfgFile)
	_ = vip.ReadInConfig()

	cfg, err := FromViperSimple(vip)
	if err != nil {
		t.Error(err)
	}

	t.Logf("\n========== Read Config and transform to cluster ==========\n%+v\n=================================\n", cfg)

	clusterCfg, err := TransformSimpleToClusterConfig(context.Background(), runtimes.Docker, cfg)
	if err != nil {
		t.Error(err)
	}

	t.Logf("\n========== Process Cluster Config (non-host network) ==========\n%+v\n=================================\n", cfg)

	clusterCfg, err = ProcessClusterConfig(*clusterCfg)
	assert.Assert(t, clusterCfg.ClusterCreateOpts.DisableLoadBalancer == false, "The load balancer should be enabled")
	assert.Assert(t, clusterCfg.ClusterCreateOpts.PrepDisableHostIPInjection == false, "The host ip injection should be enabled")

	t.Logf("\n===== Resulting Cluster Config (non-host network) =====\n%+v\n===============\n", clusterCfg)

	t.Logf("\n========== Process Cluster Config (host network) ==========\n%+v\n=================================\n", cfg)

	clusterCfg.Cluster.Network.Name = "host"
	clusterCfg, err = ProcessClusterConfig(*clusterCfg)
	assert.Assert(t, clusterCfg.ClusterCreateOpts.DisableLoadBalancer == true, "The load balancer should be disabled")
	assert.Assert(t, clusterCfg.ClusterCreateOpts.PrepDisableHostIPInjection == true, "The host ip injection should be disabled")

	t.Logf("\n===== Resulting Cluster Config (host network) =====\n%+v\n===============\n", clusterCfg)

}
