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
	"os"
	"os/exec"
	"strings"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
)

func (d Docker) GetDockerMachineIP() (string, error) {
	machine := os.ExpandEnv("$DOCKER_MACHINE_NAME")
	if machine == "" {
		l.Log().Tracef("Docker Machine not specified via DOCKER_MACHINE_NAME env var")
		return "", nil
	}

	l.Log().Debugf("Docker Machine found: %s", machine)
	dockerMachinePath, err := exec.LookPath("docker-machine")
	if err != nil {
		if err == exec.ErrNotFound {
			l.Log().Debugf("DOCKER_MACHINE_NAME env var present, but executable docker-machine not found: %+v", err)
		}
		return "", nil
	}

	out, err := exec.Command(dockerMachinePath, "ip", machine).Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf(string(exitError.Stderr))
		}
		return "", err
	}
	ipStr := strings.TrimSuffix(string(out), "\n")
	ipStr = strings.TrimSuffix(ipStr, "\r")

	return ipStr, nil
}
