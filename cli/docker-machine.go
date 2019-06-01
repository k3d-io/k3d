package run

import (
	"os"
	"os/exec"
	"strings"
)

func getDockerMachineIp() (string, error) {
	machine := os.ExpandEnv("$DOCKER_MACHINE_NAME")

	if machine == "" {
		return "", nil
	}

	dockerMachinePath, err := exec.LookPath("docker-machine")
	if err != nil {
		return "", err
	}

	out, err := exec.Command(dockerMachinePath, "ip", machine).Output()

	ipStr := strings.TrimSuffix(string(out), "\n")
	ipStr = strings.TrimSuffix(ipStr, "\r")
	return ipStr, err
}
