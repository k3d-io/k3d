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
	"bufio"
	"context"
	"fmt"
	"net/netip"
	"regexp"
	goruntime "runtime"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	"github.com/k3d-io/k3d/v5/pkg/runtimes/docker"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/pkg/util"
)

type ResolveHostCmd struct {
	Cmd        string
	LogMatcher *regexp.Regexp
}

var (
	ResolveHostCmdNSLookup = ResolveHostCmd{
		Cmd:        "nslookup %s",
		LogMatcher: regexp.MustCompile(`^Address:\s+(?P<ip>\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})$`),
	}

	ResolveHostCmdGetEnt = ResolveHostCmd{
		Cmd:        "getent ahostsv4 '%s'",
		LogMatcher: regexp.MustCompile(`(?P<ip>\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\s+STREAM.+`), // e.g. `192.168.47.4   STREAM host.docker.internal`,
	}
)

// GetHostIP returns the routable IP address to be able to access services running on the host system from inside the cluster.
// This depends on the Operating System and the chosen Runtime.
func GetHostIP(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster) (netip.Addr, error) {
	rtimeInfo, err := runtime.Info()
	if err != nil {
		return netip.Addr{}, err
	}

	l.Log().Tracef("GOOS: %s / Runtime OS: %s (%s)", goruntime.GOOS, rtimeInfo.OSType, rtimeInfo.OS)

	// Docker Runtime
	if runtime == runtimes.Docker {
		// Docker (for Desktop) on MacOS or Windows
		if docker.IsDockerDesktop(rtimeInfo.OS) {
			toolsNode, err := EnsureToolsNode(ctx, runtime, cluster)
			if err != nil {
				return netip.Addr{}, fmt.Errorf("failed to ensure that k3d-tools node is running to get host IP :%w", err)
			}

			k3dInternalIP, err := resolveHostnameFromInside(ctx, runtime, toolsNode, k3d.DefaultK3dInternalHostRecord, ResolveHostCmdGetEnt)
			if err == nil {
				return k3dInternalIP, nil
			}

			l.Log().Debugf("[GetHostIP on Docker Desktop] failed to resolve '%s' from inside the k3d-tools node: %v", k3d.DefaultK3dInternalHostRecord, err)

			dockerInternalIP, err := resolveHostnameFromInside(ctx, runtime, toolsNode, "host.docker.internal", ResolveHostCmdGetEnt)
			if err == nil {
				return dockerInternalIP, nil
			}

			l.Log().Debugf("[GetHostIP on Docker Desktop] failed to resolve 'host.docker.internal' from inside the k3d-tools node: %v", err)
		}

		// Colima
		if rtimeInfo.InfoName == "colima" {
			toolsNode, err := EnsureToolsNode(ctx, runtime, cluster)
			if err != nil {
				return netip.Addr{}, fmt.Errorf("failed to ensure that k3d-tools node is running to get host IP :%w", err)
			}

			limaIP, err := resolveHostnameFromInside(ctx, runtime, toolsNode, "host.lima.internal", ResolveHostCmdGetEnt)
			if err == nil {
				return limaIP, nil
			}

			l.Log().Debugf("[GetHostIP on colima] failed to resolve 'host.lima.internal' from inside the k3d-tools node: %v", err)
		}

		ip, err := runtime.GetHostIP(ctx, cluster.Network.Name)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("runtime failed to get host IP: %w", err)
		}
		l.Log().Infof("HostIP: using network gateway %s address", ip)

		return ip, nil
	}

	// Catch all other runtime selections
	return netip.Addr{}, fmt.Errorf("GetHostIP only implemented for the docker runtime")
}

func resolveHostnameFromInside(ctx context.Context, rtime runtimes.Runtime, node *k3d.Node, hostname string, cmd ResolveHostCmd) (netip.Addr, error) {
	errPrefix := fmt.Errorf("error resolving hostname %s from inside node %s", hostname, node.Name)

	logreader, execErr := rtime.ExecInNodeGetLogs(ctx, node, []string{"sh", "-c", fmt.Sprintf(cmd.Cmd, hostname)})

	if logreader == nil {
		if execErr != nil {
			return netip.Addr{}, fmt.Errorf("%v: %w", errPrefix, execErr)
		}
		return netip.Addr{}, fmt.Errorf("%w: failed to get logs from exec process", errPrefix)
	}

	submatches := map[string]string{}
	scanner := bufio.NewScanner(logreader)
	if scanner == nil {
		if execErr != nil {
			return netip.Addr{}, fmt.Errorf("%v: %w", errPrefix, execErr)
		}
		return netip.Addr{}, fmt.Errorf("Failed to scan logs for host IP: Could not create scanner from logreader")
	}
	if scanner != nil && execErr != nil {
		l.Log().Debugln("Exec Process Failed, but we still got logs, so we're at least trying to get the IP from there...")
		l.Log().Tracef("-> Exec Process Error was: %+v", execErr)
	}
	for scanner.Scan() {
		l.Log().Tracef("Scanning Log Line '%s'", scanner.Text())
		match := cmd.LogMatcher.FindStringSubmatch(scanner.Text())
		if len(match) == 0 {
			continue
		}
		submatches = util.MapSubexpNames(cmd.LogMatcher.SubexpNames(), match)
		l.Log().Tracef("-> Match(es): '%+v' -> Submatch(es): %+v", match, submatches)
		break
	}
	if _, ok := submatches["ip"]; !ok {
		if execErr != nil {
			l.Log().Errorln(execErr)
		}
		return netip.Addr{}, fmt.Errorf("%w: failed to read address for '%s' from command output", errPrefix, hostname)
	}

	l.Log().Debugf("Hostname '%s' resolved to address '%s' inside node %s", hostname, submatches["ip"], node.Name)

	return netip.ParseAddr(submatches["ip"])
}
