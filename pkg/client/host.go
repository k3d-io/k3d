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
package client

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"regexp"
	"runtime"

	l "github.com/rancher/k3d/v5/pkg/logger"
	rt "github.com/rancher/k3d/v5/pkg/runtimes"
	k3d "github.com/rancher/k3d/v5/pkg/types"
	"github.com/rancher/k3d/v5/pkg/util"
)

var nsLookupAddressRegexp = regexp.MustCompile(`^Address:\s+(?P<ip>\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})$`)

// GetHostIP returns the routable IP address to be able to access services running on the host system from inside the cluster.
// This depends on the Operating System and the chosen Runtime.
func GetHostIP(ctx context.Context, rtime rt.Runtime, cluster *k3d.Cluster) (net.IP, error) {

	// Docker Runtime
	if rtime == rt.Docker {

		l.Log().Tracef("Runtime GOOS: %s", runtime.GOOS)

		// "native" Docker on Linux
		if runtime.GOOS == "linux" {
			ip, err := rtime.GetHostIP(ctx, cluster.Network.Name)
			if err != nil {
				return nil, fmt.Errorf("runtime failed to get host IP: %w", err)
			}
			return ip, nil
		}

		// Docker (for Desktop) on MacOS or Windows
		if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {

			toolsNode, err := EnsureToolsNode(ctx, rtime, cluster)
			if err != nil {
				return nil, fmt.Errorf("failed to ensure that k3d-tools node is running to get host IP :%w", err)
			}

			ip, err := resolveHostnameFromInside(ctx, rtime, toolsNode, "host.docker.internal")
			if err != nil {
				return nil, fmt.Errorf("failed to resolve 'host.docker.internal' from inside the k3d-tools node: %w", err)
			}
			return ip, nil
		}

		// Catch all other GOOS cases
		return nil, fmt.Errorf("GetHostIP only implemented for Linux, MacOS (Darwin) and Windows")

	}

	// Catch all other runtime selections
	return nil, fmt.Errorf("GetHostIP only implemented for the docker runtime")

}

func resolveHostnameFromInside(ctx context.Context, rtime rt.Runtime, node *k3d.Node, hostname string) (net.IP, error) {

	logreader, execErr := rtime.ExecInNodeGetLogs(ctx, node, []string{"sh", "-c", fmt.Sprintf("nslookup %s", hostname)})

	if logreader == nil {
		if execErr != nil {
			return nil, execErr
		}
		return nil, fmt.Errorf("Failed to get logs from exec process")
	}

	submatches := map[string]string{}
	scanner := bufio.NewScanner(logreader)
	if scanner == nil {
		if execErr != nil {
			return nil, execErr
		}
		return nil, fmt.Errorf("Failed to scan logs for host IP: Could not create scanner from logreader")
	}
	if scanner != nil && execErr != nil {
		l.Log().Debugln("Exec Process Failed, but we still got logs, so we're at least trying to get the IP from there...")
		l.Log().Tracef("-> Exec Process Error was: %+v", execErr)
	}
	for scanner.Scan() {
		l.Log().Tracef("Scanning Log Line '%s'", scanner.Text())
		match := nsLookupAddressRegexp.FindStringSubmatch(scanner.Text())
		if len(match) == 0 {
			continue
		}
		l.Log().Tracef("-> Match(es): '%+v'", match)
		submatches = util.MapSubexpNames(nsLookupAddressRegexp.SubexpNames(), match)
		l.Log().Tracef(" -> Submatch(es): %+v", submatches)
		break
	}
	if _, ok := submatches["ip"]; !ok {
		if execErr != nil {
			l.Log().Errorln(execErr)
		}
		return nil, fmt.Errorf("Failed to read address for '%s' from nslookup response", hostname)
	}

	l.Log().Debugf("Hostname '%s' -> Address '%s'", hostname, submatches["ip"])

	return net.ParseIP(submatches["ip"]), nil

}
