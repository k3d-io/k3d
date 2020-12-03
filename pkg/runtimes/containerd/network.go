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
package containerd

import (
	"context"

	k3d "github.com/rancher/k3d/v3/pkg/types"
)

// CreateNetworkIfNotPresent creates a new docker network
func (d Containerd) CreateNetworkIfNotPresent(ctx context.Context, name string) (string, bool, error) {
	return "", false, nil
}

// DeleteNetwork deletes a network
func (d Containerd) DeleteNetwork(ctx context.Context, ID string) error {
	return nil
}

// ConnectNodeToNetwork connects a node to a network
func (d Containerd) ConnectNodeToNetwork(ctx context.Context, node *k3d.Node, network string) error {
	return nil
}
