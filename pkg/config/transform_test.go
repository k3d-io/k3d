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

	"github.com/rancher/k3d/v3/pkg/runtimes"
)

func TestTransformSimpleConfigToClusterConfig(t *testing.T) {
	cfgFile := "./test_assets/config_test_simple.yaml"

	cfg, err := ReadConfig(cfgFile)
	if err != nil {
		t.Error(err)
	}

	simpleCfg, ok := cfg.(SimpleConfig)
	if !ok {
		t.Error("Config is not of type SimpleConfig")
	}

	t.Logf("\n========== Read Config ==========\n%+v\n=================================\n", simpleCfg)

	clusterCfg, err := TransformSimpleToClusterConfig(context.Background(), runtimes.Docker, simpleCfg)
	if err != nil {
		t.Error(err)
	}

	t.Logf("\n===== Resulting Cluster Config =====\n%+v\n===============\n", clusterCfg)

}
