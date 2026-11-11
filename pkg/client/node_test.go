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
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/pkg/types/fixes"
)

// fakeNodeStartRuntime implements the parts of runtimes.Runtime that NodeStart
// uses until it starts the node. Calling any other method panics.
type fakeNodeStartRuntime struct {
	runtimes.Runtime
	writeErr error
	written  []string
	started  bool
	deleted  bool
}

func (r *fakeNodeStartRuntime) WriteToNode(_ context.Context, _ []byte, dest string, _ os.FileMode, _ *k3d.Node) error {
	if r.writeErr != nil {
		return r.writeErr
	}
	r.written = append(r.written, dest)
	return nil
}

func (r *fakeNodeStartRuntime) StartNode(_ context.Context, _ *k3d.Node) error {
	r.started = true
	return nil
}

func (r *fakeNodeStartRuntime) DeleteNode(_ context.Context, _ *k3d.Node) error {
	r.deleted = true
	return nil
}

func (r *fakeNodeStartRuntime) GetNodesByLabel(_ context.Context, _ map[string]string) ([]*k3d.Node, error) {
	return nil, nil
}

// enableOnlyMountsFix makes NodeStart write the k3d entrypoint scripts
// without asking the runtime which fixes are needed.
func enableOnlyMountsFix(t *testing.T) {
	t.Helper()
	oldFixes, oldAny := fixes.EnabledFixes, fixes.AnyFixEnabled
	fixes.EnabledFixes = map[fixes.K3DFixEnv]bool{fixes.EnvFixMounts: true}
	fixes.AnyFixEnabled = true
	t.Cleanup(func() { fixes.EnabledFixes, fixes.AnyFixEnabled = oldFixes, oldAny })
}

func Test_NodeStart_failsIfEntrypointCannotBeWritten(t *testing.T) {
	enableOnlyMountsFix(t)

	tests := map[string]struct {
		intent        k3d.Intent
		expectDeleted bool
	}{
		// the node was created by the same NodeRun call and never ran, so it gets cleaned up
		"node create": {intent: k3d.IntentNodeCreate, expectDeleted: true},
		// the caller rolls back the whole cluster
		"cluster create": {intent: k3d.IntentClusterCreate},
		// existing nodes must be kept
		"cluster start": {intent: k3d.IntentClusterStart},
		"node start":    {intent: k3d.IntentNodeStart},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			writeErr := errors.New("error while creating mount source path")
			runtime := &fakeNodeStartRuntime{writeErr: writeErr}
			node := &k3d.Node{Name: "k3d-test-server-0", Role: k3d.ServerRole}

			err := NodeStart(context.Background(), runtime, node, &k3d.NodeStartOpts{Intent: tc.intent})
			if err == nil {
				t.Fatal("expected NodeStart to fail, got no error")
			}
			if !errors.Is(err, writeErr) {
				t.Errorf("expected the error to wrap the write error, got: %v", err)
			}
			if !strings.Contains(err.Error(), "/bin/k3d-entrypoint.sh") {
				t.Errorf("expected the error to name the entrypoint script, got: %v", err)
			}
			if runtime.started {
				t.Error("node was started although its entrypoint script could not be written")
			}
			if runtime.deleted != tc.expectDeleted {
				t.Errorf("node deleted: got %t, expected %t", runtime.deleted, tc.expectDeleted)
			}
		})
	}
}

func Test_NodeStart_writesEntrypointBeforeStart(t *testing.T) {
	enableOnlyMountsFix(t)

	runtime := &fakeNodeStartRuntime{}
	node := &k3d.Node{Name: "k3d-test-server-0", Role: k3d.ServerRole}

	if err := NodeStart(context.Background(), runtime, node, &k3d.NodeStartOpts{Intent: k3d.IntentClusterCreate}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !runtime.started {
		t.Error("node was not started")
	}
	if len(runtime.written) == 0 || runtime.written[0] != "/bin/k3d-entrypoint.sh" {
		t.Errorf("expected /bin/k3d-entrypoint.sh to be written first, got: %v", runtime.written)
	}
}
