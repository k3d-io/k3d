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

package docker

import (
	"fmt"
	"net"
	"net/url"
	"os"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
)

type Docker struct{}

const (
	DefaultDockerSock = "/var/run/docker.sock"
)

// ID returns the identity of the runtime
func (d Docker) ID() string {
	return "docker"
}

// GetHost returns the docker daemon host
func (d Docker) GetHost() string {
	// a) docker-machine
	machineIP, err := d.GetDockerMachineIP()
	if err != nil {
		l.Log().Warnf("[Docker] Using docker-machine, but failed to get it's IP: %+v", err)
	} else if machineIP != "" {
		l.Log().Infof("[Docker] Using the docker-machine IP %s to connect to the Kubernetes API", machineIP)
		return machineIP
	} else {
		l.Log().Traceln("[Docker] Not using docker-machine")
	}

	// b) DOCKER_HOST env var
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		l.Log().Traceln("[Docker] GetHost: DOCKER_HOST empty/unset")
		info, err := d.Info()
		if err != nil {
			l.Log().Errorf("[Docker] error getting runtime information: %v", err)
			return ""
		}
		// c) Docker for Desktop (Win/Mac) and it's a local connection
		if IsDockerDesktop(info.OS) && IsLocalConnection(info.Endpoint) {
			// c.1) local DfD connection, but inside WSL, where host.docker.internal resolves to an IP, but it's not reachable
			if _, ok := os.LookupEnv("WSL_DISTRO_NAME"); ok {
				l.Log().Debugln("[Docker] wanted to use 'host.docker.internal' as docker host, but it's not reachable in WSL2")
				return ""
			}
			l.Log().Debugln("[Docker] Local DfD: using 'host.docker.internal'")
			dfdHost := "host.docker.internal"
			if _, err := net.LookupHost(dfdHost); err != nil {
				l.Log().Debugf("[Docker] wanted to use 'host.docker.internal' as docker host, but it's not resolvable locally: %v", err)
				return ""
			}
			dockerHost = fmt.Sprintf("tcp://%s", dfdHost)
		}
	}
	url, err := url.Parse(dockerHost)
	if err != nil {
		l.Log().Debugf("[Docker] GetHost: error parsing '%s' as URL: %#v", dockerHost, url)
		return ""
	}
	dockerHost = url.Host
	l.Log().Debugf("[Docker] DockerHost: '%s' (%+v)", dockerHost, url)

	return dockerHost
}

// GetRuntimePath returns the path of the docker socket
func (d Docker) GetRuntimePath() string {
	dockerSock := os.Getenv("DOCKER_SOCK")
	if dockerSock == "" {
		dockerSock = DefaultDockerSock
	}
	l.Log().Debugf("DOCKER_SOCK=%s", dockerSock)
	return dockerSock
}
