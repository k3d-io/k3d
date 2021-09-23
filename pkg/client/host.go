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
	goruntime "runtime"
	"strings"

	l "github.com/rancher/k3d/v5/pkg/logger"
	"github.com/rancher/k3d/v5/pkg/runtimes"
	k3d "github.com/rancher/k3d/v5/pkg/types"
	"github.com/rancher/k3d/v5/pkg/util"
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
func GetHostIP(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster) (net.IP, error) {

	rtimeInfo, err := runtime.Info()
	if err != nil {
		return nil, err
	}

	l.Log().Tracef("GOOS: %s / Runtime OS: %s (%s)", goruntime.GOOS, rtimeInfo.OSType, rtimeInfo.OS)

	isDockerDesktop := func(os string) bool {
		return strings.ToLower(os) == "docker desktop"
	}

	// Docker Runtime
	if runtime == runtimes.Docker {

		// "native" Docker on Linux
		if goruntime.GOOS == "linux" && rtimeInfo.OSType == "linux" {
			ip, err := runtime.GetHostIP(ctx, cluster.Network.Name)
			if err != nil {
				return nil, fmt.Errorf("runtime failed to get host IP: %w", err)
			}
			return ip, nil
		}

		// Docker (for Desktop) on MacOS or Windows
		if (rtimeInfo.OSType == "windows" || rtimeInfo.OSType == "darwin") && isDockerDesktop(rtimeInfo.OS) {

			toolsNode, err := EnsureToolsNode(ctx, runtime, cluster)
			if err != nil {
				return nil, fmt.Errorf("failed to ensure that k3d-tools node is running to get host IP :%w", err)
			}

			var ip net.IP

			ip, err = resolveHostnameFromInside(ctx, runtime, toolsNode, "host.docker.internal", ResolveHostCmdGetEnt)
			if err == nil {
				return ip, nil
			}

			l.Log().Warnf("failed to resolve 'host.docker.internal' from inside the k3d-tools node: %v", err)

			l.Log().Infof("HostIP-Fallback: using network gateway...")
			ip, err = runtime.GetHostIP(ctx, cluster.Network.Name)
			if err != nil {
				return nil, fmt.Errorf("runtime failed to get host IP: %w", err)
			}

			return ip, nil
		}

		// Catch all other GOOS cases
		return nil, fmt.Errorf("GetHostIP not implemented for Docker and the combination of k3d host '%s' / docker host '%s (%s)'", goruntime.GOOS, rtimeInfo.OSType, rtimeInfo.OS)

	}

	// Catch all other runtime selections
	return nil, fmt.Errorf("GetHostIP only implemented for the docker runtime")

}

func resolveHostnameFromInside(ctx context.Context, rtime runtimes.Runtime, node *k3d.Node, hostname string, cmd ResolveHostCmd) (net.IP, error) {

	logreader, execErr := rtime.ExecInNodeGetLogs(ctx, node, []string{"sh", "-c", fmt.Sprintf(cmd.Cmd, hostname)})

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
		match := cmd.LogMatcher.FindStringSubmatch(scanner.Text())
		if len(match) == 0 {
			continue
		}
		l.Log().Tracef("-> Match(es): '%+v'", match)
		submatches = util.MapSubexpNames(cmd.LogMatcher.SubexpNames(), match)
		l.Log().Tracef(" -> Submatch(es): %+v", submatches)
		break
	}
	if _, ok := submatches["ip"]; !ok {
		if execErr != nil {
			l.Log().Errorln(execErr)
		}
		return nil, fmt.Errorf("Failed to read address for '%s' from command output", hostname)
	}

	l.Log().Debugf("Hostname '%s' -> Address '%s'", hostname, submatches["ip"])

	return net.ParseIP(submatches["ip"]), nil

}
