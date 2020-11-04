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

package docker

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

func (d Docker) GetDockerMachineIP() (string, error) {
	machine := os.ExpandEnv("$DOCKER_MACHINE_NAME")
	dockerHostName := "host.docker.internal"

	// Option 1: use the docker-machine executable
	if machine != "" {
		dockerMachinePath, err := exec.LookPath("docker-machine")
		if err != nil {
			return "", err
		}

		out, err := exec.Command(dockerMachinePath, "ip", machine).Output()
		if err != nil {
			log.Printf("Error executing 'docker-machine ip'")

			if exitError, ok := err.(*exec.ExitError); ok {
				log.Printf("%s", string(exitError.Stderr))
			}
			return "", err
		}
		ipStr := strings.TrimSuffix(string(out), "\n")
		ipStr = strings.TrimSuffix(ipStr, "\r")

		return ipStr, nil
	}

	// Option 2: Try to lookup "host.docker.internal"
	log.Debugf("Docker Machine not found, trying to lookup '%s' instead...", dockerHostName)
	addrs, err := net.LookupHost(dockerHostName)
	if err != nil {
		log.Debugf("Lookup of Host %s failed: %+v", dockerHostName, err)
		return "", fmt.Errorf("Failed to get IP of Docker VM")
	}
	if len(addrs) == 0 {
		log.Debugf("Lookup of Host %s didn't return any addresses", dockerHostName)
		return "", fmt.Errorf("Failed to get IP of Docker VM")
	}
	log.Debugf("Addresses returned for %s: %+v", dockerHostName, addrs)
	return addrs[0], nil
}
