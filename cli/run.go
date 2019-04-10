package run

import (
	"log"
	"os"
	"os/exec"
)

// runCommand accepts the name and args and runs the specified command
func runCommand(verbose bool, name string, args ...string) error {
	if verbose {
		log.Printf("Running command: %+v", append([]string{name}, args...))
	}
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
