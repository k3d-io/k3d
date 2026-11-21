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
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

// fakeToolsRuntime implements just enough of runtimes.Runtime to exercise importWithToolsNode.
// Methods that are not overridden are not expected to be called and will panic on the embedded nil interface.
type fakeToolsRuntime struct {
	runtimes.Runtime

	// execErrByNode returns the error that ExecInNode should produce for a given node name
	execErrByNode map[string]error
	// copyErr is returned by every CopyToNode call
	copyErr error

	mutex     sync.Mutex
	execCalls []string
}

func (f *fakeToolsRuntime) GetNode(_ context.Context, node *k3d.Node) (*k3d.Node, error) {
	// pretend a tools node is already up, so that EnsureToolsNode short-circuits
	return &k3d.Node{Name: node.Name, State: k3d.NodeState{Running: true}}, nil
}

func (f *fakeToolsRuntime) ExecInNode(_ context.Context, node *k3d.Node, cmd []string) error {
	f.mutex.Lock()
	f.execCalls = append(f.execCalls, node.Name+": "+strings.Join(cmd, " "))
	f.mutex.Unlock()
	return f.execErrByNode[node.Name]
}

func (f *fakeToolsRuntime) CopyToNode(_ context.Context, _ string, _ string, _ *k3d.Node) error {
	return f.copyErr
}

func (f *fakeToolsRuntime) DeleteNode(_ context.Context, _ *k3d.Node) error {
	return nil
}

func (f *fakeToolsRuntime) ExecInNodeWithStdin(_ context.Context, node *k3d.Node, _ []string, stdin io.ReadCloser) error {
	// Consume one chunk so that the writer side of the pipe can make progress.
	// We deliberately do not read until EOF: loadImageFromStream never closes the pipe writers,
	// so a full drain would block forever (just like `ctr image import -` waiting for more input).
	buf := make([]byte, 4096)
	if _, err := stdin.Read(buf); err != nil && err != io.EOF {
		return err
	}
	return f.execErrByNode[node.Name]
}

func testCluster() *k3d.Cluster {
	return &k3d.Cluster{
		Name: "test",
		Nodes: []*k3d.Node{
			{Name: "k3d-test-server-0", Role: k3d.ServerRole},
			{Name: "k3d-test-agent-0", Role: k3d.AgentRole},
			{Name: "k3d-test-serverlb", Role: k3d.LoadBalancerRole},
		},
	}
}

func Test_importWithToolsNode_returnsErrorWhenImportIntoNodeFails(t *testing.T) {
	// given: importing into one of the cluster nodes fails
	runtime := &fakeToolsRuntime{
		execErrByNode: map[string]error{
			"k3d-test-agent-0": errors.New("ctr: image import failed"),
		},
	}

	// when
	err := importWithToolsNode(context.Background(), runtime, testCluster(), []string{"alpine:latest"}, nil, k3d.ImageImportOpts{})

	// then
	if err == nil {
		t.Fatal("expected an error when importing images into a node fails, but got nil")
	}
	if !strings.Contains(err.Error(), "k3d-test-agent-0") {
		t.Errorf("expected error to name the failing node, got: %v", err)
	}
}

func Test_importWithToolsNode_returnsErrorWhenTarballCopyFails(t *testing.T) {
	// given: copying the image tarball to the tools node fails
	runtime := &fakeToolsRuntime{
		execErrByNode: map[string]error{},
		copyErr:       errors.New("no space left on device"),
	}

	// when
	err := importWithToolsNode(context.Background(), runtime, testCluster(), nil, []string{"/tmp/images.tar"}, k3d.ImageImportOpts{})

	// then
	if err == nil {
		t.Fatal("expected an error when copying the image tarball fails, but got nil")
	}
	if !strings.Contains(err.Error(), "/tmp/images.tar") {
		t.Errorf("expected error to name the tarball that could not be copied, got: %v", err)
	}
}

func Test_ImageImportIntoClusterMulti_succeedsWhenReadingImageFromStdin(t *testing.T) {
	// given: an image tarball on stdin that imports successfully into every node
	tarball, err := os.CreateTemp(t.TempDir(), "images-*.tar")
	if err != nil {
		t.Fatalf("failed to create temporary tarball: %v", err)
	}
	if _, err := tarball.WriteString("not a real tarball, but the runtime is faked anyway"); err != nil {
		t.Fatalf("failed to write temporary tarball: %v", err)
	}
	if _, err := tarball.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("failed to rewind temporary tarball: %v", err)
	}

	originalStdin := os.Stdin
	os.Stdin = tarball
	t.Cleanup(func() { os.Stdin = originalStdin })

	runtime := &fakeToolsRuntime{execErrByNode: map[string]error{}}

	// when
	err = ImageImportIntoClusterMulti(context.Background(), runtime, []string{"-"}, testCluster(), k3d.ImageImportOpts{})

	// then
	if err != nil {
		t.Errorf("expected no error when the import from stdin succeeds, got: %v", err)
	}
}
