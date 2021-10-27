/*
Copyright © 2020-2021 The k3d Author(s)

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

	"github.com/rancher/k3d/v5/pkg/runtimes"
	k3d "github.com/rancher/k3d/v5/pkg/types"

	l "github.com/rancher/k3d/v5/pkg/logger"
)

type WriteFileAction struct {
	Runtime runtimes.Runtime
	Content []byte
	Dest    string
	Mode    os.FileMode
}

func (act WriteFileAction) Run(ctx context.Context, node *k3d.Node) error {
	return act.Runtime.WriteToNode(ctx, act.Content, act.Dest, act.Mode, node)
}

type RewriteFileAction struct {
	Runtime     runtimes.Runtime
	Path        string
	RewriteFunc func([]byte) ([]byte, error)
	Mode        os.FileMode
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

	return act.Runtime.WriteToNode(ctx, file, act.Path, act.Mode, node)

}
