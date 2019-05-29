package run

import (
	"fmt"
	"os"
	"os/exec"
)

func bashShell(cluster string, command string) error {
	kubeConfigPath, err := getKubeConfig(cluster)
	if err != nil {
		return err
	}

	subShell := os.ExpandEnv("$__K3D_CLUSTER__")
	if len(subShell) > 0 {
		return fmt.Errorf("Error: Already in subshell of cluster %s", subShell)
	}

	bashPath, err := exec.LookPath("bash")
	if err != nil {
		return err
	}

	cmd := exec.Command(bashPath, "--noprofile", "--norc")

	if len(command) > 0 {
		cmd.Args = append(cmd.Args, "-c", command)

	}

	// Set up stdio
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	// Set up Promot
	setPS1 := fmt.Sprintf("PS1=[%s}%s", cluster, os.Getenv("PS1"))

	// Set up KUBECONFIG
	setKube := fmt.Sprintf("KUBECONFIG=%s", kubeConfigPath)

	// Declare subshell
	subShell = fmt.Sprintf("__K3D_CLUSTER__=%s", cluster)

	newEnv := append(os.Environ(), setPS1, setKube, subShell)

	cmd.Env = newEnv

	return cmd.Run()
}
