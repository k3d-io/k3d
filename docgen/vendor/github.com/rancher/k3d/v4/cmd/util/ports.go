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
	"regexp"
	"strconv"

	"github.com/docker/go-connections/nat"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	"github.com/rancher/k3d/v4/pkg/util"
	log "github.com/sirupsen/logrus"
)

var apiPortRegexp = regexp.MustCompile(`^(?P<hostref>(?P<hostip>\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})|(?P<hostname>\S+):)?(?P<port>(\d{1,5}|random))$`)

// ParsePortExposureSpec parses/validates a string to create an exposePort struct from it
func ParsePortExposureSpec(exposedPortSpec, internalPort string) (*k3d.ExposureOpts, error) {

	match := apiPortRegexp.FindStringSubmatch(exposedPortSpec)

	if len(match) == 0 {
		return nil, fmt.Errorf("Failed to parse Port Exposure specification '%s': Format must be [(HostIP|HostName):]HostPort", exposedPortSpec)
	}

	submatches := util.MapSubexpNames(apiPortRegexp.SubexpNames(), match)

	// no port specified (or not matched via regex)
	if submatches["port"] == "" {
		return nil, fmt.Errorf("Failed to find port in Port Exposure spec '%s'", exposedPortSpec)
	}

	api := &k3d.ExposureOpts{}

	// check if there's a host reference
	if submatches["hostname"] != "" {
		log.Tracef("Port Exposure: found hostname: %s", submatches["hostname"])
		addrs, err := net.LookupHost(submatches["hostname"])
		if err != nil {
			return nil, fmt.Errorf("Failed to lookup host '%s' specified for Port Exposure: %+v", submatches["hostname"], err)
		}
		api.Host = submatches["hostname"]
		submatches["hostip"] = addrs[0] // set hostip to the resolved address
	}

	realPortString := ""

	if submatches["hostip"] == "" {
		submatches["hostip"] = k3d.DefaultAPIHost
	}

	// start with the IP, if there is any
	if submatches["hostip"] != "" {
		realPortString += submatches["hostip"] + ":"
	}

	// port: get a free one if there's none defined or set to random
	if submatches["port"] == "" || submatches["port"] == "random" {
		log.Debugf("Port Exposure Mapping didn't specify hostPort, choosing one randomly...")
		freePort, err := GetFreePort()
		if err != nil || freePort == 0 {
			log.Warnf("Failed to get random free port: %+v", err)
			log.Warnf("Falling back to internal port %s (may be blocked though)...", internalPort)
			submatches["port"] = internalPort
		} else {
			submatches["port"] = strconv.Itoa(freePort)
			log.Debugf("Got free port for Port Exposure: '%d'", freePort)
		}
	}

	realPortString += fmt.Sprintf("%s:%s/tcp", submatches["port"], internalPort)

	portMapping, err := nat.ParsePortSpec(realPortString)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse port spec for Port Exposure '%s': %+v", realPortString, err)
	}

	api.Port = portMapping[0].Port // there can be only one due to our regexp
	api.Binding = portMapping[0].Binding

	return api, nil

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
