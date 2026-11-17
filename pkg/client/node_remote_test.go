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
	"errors"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	runtimeTypes "github.com/k3d-io/k3d/v5/pkg/runtimes/types"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

type fakeRemoteNodeRuntime struct {
	gateways      map[string]netip.Addr
	askedNetworks []string
}

func (f *fakeRemoteNodeRuntime) Info() (*runtimeTypes.RuntimeInfo, error) {
	return &runtimeTypes.RuntimeInfo{Name: "fake"}, nil
}

func (f *fakeRemoteNodeRuntime) GetHostIP(_ context.Context, network string) (netip.Addr, error) {
	f.askedNetworks = append(f.askedNetworks, network)
	ip, ok := f.gateways[network]
	if !ok {
		return netip.Addr{}, errors.New("no such network")
	}
	return ip, nil
}

func TestRemoteNodeEnvironmentInfo(t *testing.T) {
	gateways := map[string]netip.Addr{
		"bridge":      netip.MustParseAddr("172.17.0.1"),
		"k3d-network": netip.MustParseAddr("172.20.0.1"),
	}

	tests := map[string]struct {
		networks     []string
		wantGateway  netip.Addr
		wantNetworks []string
		wantErr      bool
	}{
		"no network uses the default bridge": {
			wantGateway:  gateways["bridge"],
			wantNetworks: []string{"bridge"},
		},
		"first network given": {
			networks:     []string{"k3d-network", "bridge"},
			wantGateway:  gateways["k3d-network"],
			wantNetworks: []string{"k3d-network"},
		},
		"host network has no gateway": {
			networks: []string{"host"},
		},
		"unknown network": {
			networks:     []string{"nope"},
			wantNetworks: []string{"nope"},
			wantErr:      true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			rt := &fakeRemoteNodeRuntime{gateways: gateways}
			envInfo, err := remoteNodeEnvironmentInfo(context.Background(), rt, &k3d.Node{Networks: tc.networks})
			assert.Equal(t, tc.wantNetworks, rt.askedNetworks)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantGateway, envInfo.HostGateway)
			assert.Equal(t, "fake", envInfo.RuntimeInfo.Name)
		})
	}
}

// A server node that joins a remote cluster has no KubeAPI (#1485)
func TestPatchServerSpecWithoutKubeAPI(t *testing.T) {
	node := &k3d.Node{Role: k3d.ServerRole, RuntimeLabels: map[string]string{}}
	require.NoError(t, patchServerSpec(node, nil))
	assert.Equal(t, []string{"server"}, node.Cmd)
	assert.Empty(t, node.Args)
	assert.Empty(t, node.RuntimeLabels)
}

func TestPatchServerSpecWithKubeAPI(t *testing.T) {
	node := &k3d.Node{
		Role:          k3d.ServerRole,
		RuntimeLabels: map[string]string{},
		ServerOpts:    k3d.ServerOpts{KubeAPI: &k3d.ExposureOpts{Host: "k3d.example"}},
	}
	node.ServerOpts.KubeAPI.Binding.HostIP = "0.0.0.0"
	node.ServerOpts.KubeAPI.Binding.HostPort = "6443"
	require.NoError(t, patchServerSpec(node, nil))
	assert.Equal(t, []string{"--tls-san", "k3d.example"}, node.Args)
	assert.Equal(t, map[string]string{
		k3d.LabelServerAPIHostIP: "0.0.0.0",
		k3d.LabelServerAPIHost:   "k3d.example",
		k3d.LabelServerAPIPort:   "6443",
	}, node.RuntimeLabels)
}
