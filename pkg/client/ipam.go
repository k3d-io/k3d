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
	"fmt"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
	k3drt "github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"go4.org/netipx"

	"net/netip"
)

// GetIP checks a given network for a free IP and returns it, if possible
func GetIP(ctx context.Context, runtime k3drt.Runtime, network *k3d.ClusterNetwork) (netip.Addr, error) {
	network, err := runtime.GetNetwork(ctx, network)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("runtime failed to get network '%s': %w", network.Name, err)
	}

	var ipsetbuilder netipx.IPSetBuilder

	ipsetbuilder.AddPrefix(network.IPAM.IPPrefix)

	for _, ipused := range network.IPAM.IPsUsed {
		ipsetbuilder.Remove(ipused)
	}

	// exclude first and last address
	ipsetbuilder.Remove(network.IPAM.IPPrefix.Addr())
	ipsetbuilder.Remove(netipx.PrefixLastIP(network.IPAM.IPPrefix))

	ipset, err := ipsetbuilder.IPSet()
	if err != nil {
		return netip.Addr{}, err
	}

	ip := ipset.Ranges()[0].From()

	l.Log().Debugf("Found free IP %s in network %s", ip.String(), network.Name)

	return ip, nil
}
