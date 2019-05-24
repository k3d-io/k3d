package run

import (
	"fmt"
	"os"
	"os/exec"
)

func bashShell(cluster string) error {
	kubeConfigPath, err := getKubeConfig(cluster)
	if err != nil {
		return err
	}

	bashPath, err := exec.LookPath("bash")
	if err != nil {
		return err
	}

	cmd := exec.Command(bashPath, "--noprofile", "--norc")

	// Set up stdio
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	// Set up Promot
	setPS1 := fmt.Sprintf("PS1=[%s}%s", cluster, os.Getenv("PS1"))

	// Set up KUBECONFIG
	setKube := fmt.Sprintf("KUBECONFIG=%s", kubeConfigPath)
	newEnv := append(os.Environ(), setPS1, setKube)

	cmd.Env = newEnv

	return cmd.Run()
}
