/*
Copyright © 2026 The k3d Author(s)

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
	"net/netip"
	"testing"

	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	"github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/pkg/types/fixes"
)

func Test_enableFixes_MustSucceed(t *testing.T) {
	// Reset cached fixes state for testing
	prevEnabledFixes := fixes.EnabledFixes
	prevAnyFixEnabled := fixes.AnyFixEnabled
	defer func() {
		fixes.EnabledFixes = prevEnabledFixes
		fixes.AnyFixEnabled = prevAnyFixEnabled
	}()

	// Force-enable fixes
	fixes.EnabledFixes = map[fixes.K3DFixEnv]bool{
		fixes.EnvFixDNS:      true,
		fixes.EnvFixCgroupV2: true,
		fixes.EnvFixMounts:   true,
	}
	fixes.AnyFixEnabled = true

	gw, err := netip.ParseAddr("192.168.1.1")
	if err != nil {
		t.Fatalf("failed to parse test gateway IP: %v", err)
	}

	node := &types.Node{
		Name: "test-node",
		Role: types.ServerRole,
	}

	nodeStartOpts := &types.NodeStartOpts{
		EnvironmentInfo: &types.EnvironmentInfo{
			HostGateway: gw,
		},
	}

	ctx := context.Background()
	var mockRuntime runtimes.Runtime

	if err := enableFixes(ctx, mockRuntime, node, nodeStartOpts); err != nil {
		t.Fatalf("enableFixes failed: %v", err)
	}

	if len(nodeStartOpts.NodeHooks) == 0 {
		t.Fatal("expected node hooks to be populated, got none")
	}

	for i, hook := range nodeStartOpts.NodeHooks {
		if !hook.MustSucceed {
			t.Errorf("hook %d (%s) expected MustSucceed: true, got false", i, hook.Action.Info())
		}
		if hook.Stage != types.LifecycleStagePreStart {
			t.Errorf("hook %d expected Stage: preStart, got %s", i, hook.Stage)
		}
	}
}
