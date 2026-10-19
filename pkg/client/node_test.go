/*
Copyright © 2020-2025 The k3d Author(s)

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
	"strings"
	"testing"
	"time"

	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	runtimeTypes "github.com/k3d-io/k3d/v5/pkg/runtimes/types"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

// logStreamRuntime stands in for a runtime whose log endpoint refuses the
// first few opens. Only the two methods NodeWaitForLogMessage uses are real;
// the embedded interface is nil, so any other call would panic and show up.
type logStreamRuntime struct {
	runtimes.Runtime
	failOpens int
	running   bool
	opens     int
	// When set, the status probe outlives the context it was given: it calls
	// the hook, waits for the context to end, and answers with the context's
	// error, the way a probe that runs past the caller's deadline does.
	outliveContext func()
}

func (r *logStreamRuntime) GetNodeLogs(_ context.Context, _ *k3d.Node, _ time.Time, _ *runtimeTypes.NodeLogsOpts) (io.ReadCloser, error) {
	r.opens++
	if r.opens <= r.failOpens {
		return nil, errors.New("failed to obtain logs: unable to open a handle to the library")
	}
	return io.NopCloser(strings.NewReader("k3s is up and running\n")), nil
}

func (r *logStreamRuntime) GetNodeStatus(ctx context.Context, _ *k3d.Node) (bool, string, error) {
	if r.outliveContext != nil {
		r.outliveContext()
		<-ctx.Done()
		return false, "", ctx.Err()
	}
	return r.running, "running", nil
}

func Test_NodeWaitForLogMessage_RetriesFailedStreamOpen(t *testing.T) {
	rt := &logStreamRuntime{failOpens: 2, running: true}
	node := &k3d.Node{Name: "k3d-test-server-0", Role: k3d.ServerRole}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := NodeWaitForLogMessage(ctx, rt, node, "k3s is up and running", time.Time{}); err != nil {
		t.Fatalf("expected the wait to succeed after retrying the stream open, got: %v", err)
	}
	if rt.opens != 3 {
		t.Errorf("expected 3 stream opens (2 refused, 1 served), got %d", rt.opens)
	}
}

func Test_NodeWaitForLogMessage_NoRetryWhenNodeNotRunning(t *testing.T) {
	rt := &logStreamRuntime{failOpens: 1, running: false}
	node := &k3d.Node{Name: "k3d-test-server-0", Role: k3d.ServerRole}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := NodeWaitForLogMessage(ctx, rt, node, "k3s is up and running", time.Time{})
	if err == nil {
		t.Fatal("expected the wait to fail for a node that is not running")
	}
	if rt.opens != 1 {
		t.Errorf("expected exactly 1 stream open with no retry, got %d", rt.opens)
	}
}

// A context that ends while the status probe runs has to come back as the
// context's error, not as the log-open error it interrupted: callers such as
// UpdateLoadbalancerConfig choose their recovery with errors.Is against
// context.DeadlineExceeded.
func Test_NodeWaitForLogMessage_ReportsContextErrorFromStatusProbe(t *testing.T) {
	node := &k3d.Node{Name: "k3d-test-server-0", Role: k3d.ServerRole}

	t.Run("deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		rt := &logStreamRuntime{failOpens: 1, running: true, outliveContext: func() {}}

		err := NodeWaitForLogMessage(ctx, rt, node, "k3s is up and running", time.Time{})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected context.DeadlineExceeded from a probe that ran past the deadline, got: %v", err)
		}
	})

	t.Run("canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		rt := &logStreamRuntime{failOpens: 1, running: true, outliveContext: cancel}

		err := NodeWaitForLogMessage(ctx, rt, node, "k3s is up and running", time.Time{})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled from a probe that outlived the context, got: %v", err)
		}
	})
}
