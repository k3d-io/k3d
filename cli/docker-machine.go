package run

import (
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
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
	if err != nil {
		logrus.Printf("Error executing 'docker-machine ip'")

		if exitError, ok := err.(*exec.ExitError); ok {
			logrus.Printf("%s", string(exitError.Stderr))
		}
		return "", err
	}
	ipStr := strings.TrimSuffix(string(out), "\n")
	ipStr = strings.TrimSuffix(ipStr, "\r")

	return ipStr, nil
}
