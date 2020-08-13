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
package util

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	k3d "github.com/rancher/k3d/v3/pkg/types"
	log "github.com/sirupsen/logrus"
)

// ParseExposePort parses/validates a string to create an exposePort struct from it
func ParseExposePort(portString string) (k3d.ExposePort, error) {

	var exposePort k3d.ExposePort

	split := strings.Split(portString, ":")
	if len(split) > 2 {
		log.Errorln("Failed to parse API Port specification")
		return exposePort, fmt.Errorf("api-port format error")
	}

	if len(split) == 1 {
		exposePort = k3d.ExposePort{Port: split[0]}
	} else {
		// Make sure 'host' can be resolved to an IP address
		addrs, err := net.LookupHost(split[0])
		if err != nil {
			return exposePort, err
		}
		exposePort = k3d.ExposePort{Host: split[0], HostIP: addrs[0], Port: split[1]}
	}

	// Verify 'port' is an integer and within port ranges
	if exposePort.Port == "" || exposePort.Port == "random" {
		log.Debugf("API-Port Mapping didn't specify hostPort, choosing one randomly...")
		freePort, err := GetFreePort()
		if err != nil || freePort == 0 {
			log.Warnf("Failed to get random free port:\n%+v", err)
			log.Warnf("Falling back to default port %s (may be blocked though)...", k3d.DefaultAPIPort)
			exposePort.Port = k3d.DefaultAPIPort
		} else {
			exposePort.Port = strconv.Itoa(freePort)
			log.Debugf("Got free port for API: '%d'", freePort)
		}
	}
	p, err := strconv.Atoi(exposePort.Port)
	if err != nil {
		log.Errorln("Failed to parse port mapping")
		return exposePort, err
	}

	if p < 0 || p > 65535 {
		log.Errorln("Failed to parse API Port specification")
		return exposePort, fmt.Errorf("Port value '%d' out of range", p)
	}

	return exposePort, nil

}

// ValidatePortMap validates a port mapping
func ValidatePortMap(portmap string) (string, error) {
	return portmap, nil // TODO: ValidatePortMap: add validation of port mapping
}

// GetFreePort tries to fetch an open port from the OS-Kernel
func GetFreePort() (int, error) {
	tcpAddress, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		log.Errorln("Failed to resolve address")
		return 0, err
	}

	tcpListener, err := net.ListenTCP("tcp", tcpAddress)
	if err != nil {
		log.Errorln("Failed to create TCP Listener")
		return 0, err
	}
	defer tcpListener.Close()

	return tcpListener.Addr().(*net.TCPAddr).Port, nil
}
