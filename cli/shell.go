package run

import (
	"fmt"
	"os"
	"os/exec"
	"path"
)

var shells = map[string]map[string][]string{
	"bash": {
		"options": []string{
			"--noprofile", // don't load .profile/.bash_profile
			"--norc",      // don't load .bashrc
		},
	},
	"zsh": {
		"options": []string{
			"--no-rcs", // don't load .zshrc
		},
	},
}

// shell
func shell(cluster, shell, command string) error {

	// check if the selected shell is supported
	if shell == "auto" {
		shell = path.Base(os.Getenv("SHELL"))
	}

	supported := false
	for supportedShell := range shells {
		if supportedShell == shell {
			supported = true
		}
	}
	if !supported {
		return fmt.Errorf("ERROR: selected shell [%s] is not supported", shell)
	}

	// get kubeconfig for selected cluster
	kubeConfigPath, err := getKubeConfig(cluster)
	if err != nil {
		return err
	}

	// check if we're already in a subshell
	subShell := os.ExpandEnv("$__K3D_CLUSTER__")
	if len(subShell) > 0 {
		return fmt.Errorf("Error: Already in subshell of cluster %s", subShell)
	}

	// get path of shell executable
	shellPath, err := exec.LookPath(shell)
	if err != nil {
		return err
	}

	// set shell specific options (command line flags)
	shellOptions := shells[shell]["options"]

	cmd := exec.Command(shellPath, shellOptions...)

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
