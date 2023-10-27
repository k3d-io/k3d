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
package client

import (
	"context"
	"strings"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

func TestRegistryGenerateLocalRegistryHostingConfigMapYAML(t *testing.T) {
	var err error

	expectedYAMLString := `apiVersion: v1
data:
  localRegistryHosting.v1: |
    help: https://k3d.io/stable/usage/registries/#using-a-local-registry
    host: test-host:5432
    hostFromClusterNetwork: test-host:1234
    hostFromContainerRuntime: test-host:1234
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
	`

	reg := &k3d.Registry{
		Host: "test-host",
	}
	reg.ExposureOpts.Host = "test-host"
	reg.ExposureOpts.Port = nat.Port("1234/tcp")
	reg.ExposureOpts.Binding.HostPort = "5432"

	regs := []*k3d.Registry{reg}

	cm, err := RegistryGenerateLocalRegistryHostingConfigMapYAML(context.Background(), runtimes.Docker, regs)
	if err != nil {
		t.Error(err)
	}

	if !(strings.TrimSpace(string(cm)) == strings.TrimSpace(expectedYAMLString)) {
		t.Errorf("Computed configmap\n-> Actual:\n%s\n  does not match expected YAML\n-> Expected:\n%s", strings.TrimSpace(string(cm)), strings.TrimSpace(expectedYAMLString))
	}
}
