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
package actions

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/pkg/util"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
)

// WriteFileAction writes a file inside into the node filesystem
type WriteFileAction struct {
	Runtime     runtimes.Runtime
	Content     []byte
	Dest        string
	Mode        os.FileMode
	Description string
}

func (act WriteFileAction) Run(ctx context.Context, node *k3d.Node) error {
	return act.Runtime.WriteToNode(ctx, act.Content, act.Dest, act.Mode, node)
}

func (act WriteFileAction) Name() string {
	return "WriteFileAction"
}

func (act WriteFileAction) Info() string {
	if act.Description == "" {
		act.Description = "<no description>"
	}
	return fmt.Sprintf("[%s] Writing %d bytes to %s (mode %s): %s", act.Name(), len(act.Content), act.Dest, act.Mode.String(), act.Description)
}

// RewriteFileAction takes an existing file from the node filesystem and rewrites it using a specified rewrite function
type RewriteFileAction struct {
	Runtime     runtimes.Runtime
	Path        string
	RewriteFunc func([]byte) ([]byte, error)
	Mode        os.FileMode
	Description string
	Opts        RewriteFileActionOpts
}

type RewriteFileActionOpts struct {
	NoCopy bool // do not copy over the file, but rather overwrite the contents of the target (e.g. to avoid "Resource busy" error on /etc/hosts)
}

func (act RewriteFileAction) Name() string {
	return "RewriteFileAction"
}

func (act RewriteFileAction) Info() string {
	if act.Description == "" {
		act.Description = "<no description>"
	}
	return fmt.Sprintf("[%s] Rewriting file at %s (mode %s) using custom function: %s", act.Name(), act.Path, act.Mode.String(), act.Description)
}

func (act RewriteFileAction) Run(ctx context.Context, node *k3d.Node) error {
	reader, err := act.Runtime.ReadFromNode(ctx, act.Path, node)
	if err != nil {
		return fmt.Errorf("runtime failed to read '%s' from node '%s': %w", act.Path, node.Name, err)
	}
	defer reader.Close()

	file, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	file = bytes.Trim(file[512:], "\x00") // trim control characters, etc.

	file, err = act.RewriteFunc(file)
	if err != nil {
		return fmt.Errorf("error while rewriting %s in %s: %w", act.Path, node.Name, err)
	}

	l.Log().Tracef("Rewritten:\n%s", string(file))

	// default way: copy over file
	if !act.Opts.NoCopy {
		return act.Runtime.WriteToNode(ctx, file, act.Path, act.Mode, node)
	}

	// non-default: overwrite file contents
	tmpPath := fmt.Sprintf(
		"/tmp/%s-%s",
		strings.ReplaceAll(act.Path, "/", "-"),
		util.GenerateRandomString(20),
	)

	if err := act.Runtime.WriteToNode(ctx, file, tmpPath, act.Mode, node); err != nil {
		return fmt.Errorf("error creating temp file %s: %w", tmpPath, err)
	}

	cmd := fmt.Sprintf("cat %s > %s", tmpPath, act.Path)

	if err := act.Runtime.ExecInNode(ctx, node, []string{"sh", "-c", cmd}); err != nil {
		return fmt.Errorf("error overwriting contents of %s: %w", act.Path, err)
	}

	return nil
}

// ExecAction executes some command inside the node
type ExecAction struct {
	Runtime     runtimes.Runtime
	Command     []string
	Retries     int
	Description string
}

func (act ExecAction) Name() string {
	return "ExecAction"
}

func (act ExecAction) Info() string {
	if act.Description == "" {
		act.Description = "<no description>"
	}
	return fmt.Sprintf("[%s] Executing `%s` (%d retries): %s", act.Name(), act.Command, act.Retries, act.Description)
}

func (act ExecAction) Run(ctx context.Context, node *k3d.Node) error {
	for i := 0; i <= act.Retries; i++ {
		l.Log().Tracef("ExecAction (%s in %s) try %d/%d", act.Command, node.Name, i+1, act.Retries+1)
		logreader, err := act.Runtime.ExecInNodeGetLogs(ctx, node, act.Command)
		if err != nil {
			if logreader != nil {
				logs, logerr := io.ReadAll(logreader)
				if logerr != nil {
					err = fmt.Errorf("%w: <failed to get logs> (%v)", err, logerr)
				} else {
					err = fmt.Errorf("%w: Logs from failed exec process below:\n%s", err, string(logs))
				}
			} else {
				err = fmt.Errorf("%w: <no logreader returned>", err)
			}
			return fmt.Errorf("error executing hook %s in node %s: %w", act.Name(), node.Name, err)
		}
	}
	return nil
}
