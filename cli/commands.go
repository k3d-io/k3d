package run

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/urfave/cli"
)

// CheckTools checks if the installed tools work correctly
func CheckTools(c *cli.Context) error {
	log.Print("Checking docker...")
	cmd := "docker"
	args := []string{"version"}
	if err := runCommand(true, cmd, args...); err != nil {
		log.Fatalf("Checking docker: FAILED")
		return err
	}
	log.Println("Checking docker: SUCCESS")
	return nil
}

// CreateCluster creates a new single-node cluster container and initializes the cluster directory
func CreateCluster(c *cli.Context) error {
	if c.IsSet("timeout") && !c.IsSet("wait") {
		return errors.New("--wait flag is not specified")
	}
	port := fmt.Sprintf("%s:%s", c.String("port"), c.String("port"))
	image := fmt.Sprintf("rancher/k3s:%s", c.String("version"))
	cmd := "docker"
	args := []string{
		"run",
		"--name", c.String("name"),
		"-e", "K3S_KUBECONFIG_OUTPUT=/output/kubeconfig.yaml",
		"--publish", port,
		"--privileged",
	}
	extraArgs := []string{}
	if c.IsSet("volume") {
		extraArgs = append(extraArgs, "--volume", c.String("volume"))
	}
	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	}
	args = append(args,
		"-d",
		image,
		"server",                                // cmd
		"--https-listen-port", c.String("port"), //args
	)
	log.Printf("Creating cluster [%s]", c.String("name"))
	if err := runCommand(true, cmd, args...); err != nil {
		log.Fatalf("FAILURE: couldn't create cluster [%s] -> %+v", c.String("name"), err)
		return err
	}

	start := time.Now()
	timeout := time.Duration(c.Int("timeout")) * time.Second
	for c.IsSet("wait") {
		if timeout != 0 && !time.Now().After(start.Add(timeout)) {
			err := DeleteCluster(c)
			if err != nil {
				return err
			}
			return errors.New("Cluster timeout expired")
		}
		cmd := "docker"
		args = []string{
			"logs",
			c.String("name"),
		}
		prog := exec.Command(cmd, args...)
		output, err := prog.CombinedOutput()
		if err != nil {
			return err
		}
		if strings.Contains(string(output), "Running kubelet") {
			break
		}

		time.Sleep(1 * time.Second)
	}

	createClusterDir(c.String("name"))
	log.Printf("SUCCESS: created cluster [%s]", c.String("name"))
	log.Printf(`You can now use the cluster with:

export KUBECONFIG="$(%s get-kubeconfig --name='%s')"
kubectl cluster-info`, os.Args[0], c.String("name"))
	return nil
}

// DeleteCluster removes the cluster container and its cluster directory
func DeleteCluster(c *cli.Context) error {
	cmd := "docker"
	args := []string{"rm", c.String("name")}
	log.Printf("Deleting cluster [%s]", c.String("name"))
	if err := runCommand(true, cmd, args...); err != nil {
		log.Printf("WARNING: couldn't delete cluster [%s], trying a force remove now.", c.String("name"))
		args = append(args, "-f")
		if err := runCommand(true, cmd, args...); err != nil {
			log.Fatalf("FAILURE: couldn't delete cluster [%s] -> %+v", c.String("name"), err)
			return err
		}
	}
	deleteClusterDir(c.String("name"))
	log.Printf("SUCCESS: deleted cluster [%s]", c.String("name"))
	return nil
}

// StopCluster stops a running cluster container (restartable)
func StopCluster(c *cli.Context) error {
	cmd := "docker"
	args := []string{"stop", c.String("name")}
	log.Printf("Stopping cluster [%s]", c.String("name"))
	if err := runCommand(true, cmd, args...); err != nil {
		log.Fatalf("FAILURE: couldn't stop cluster [%s] -> %+v", c.String("name"), err)
		return err
	}
	log.Printf("SUCCESS: stopped cluster [%s]", c.String("name"))
	return nil
}

// StartCluster starts a stopped cluster container
func StartCluster(c *cli.Context) error {
	cmd := "docker"
	args := []string{"start", c.String("name")}
	log.Printf("Starting cluster [%s]", c.String("name"))
	if err := runCommand(true, cmd, args...); err != nil {
		log.Fatalf("FAILURE: couldn't start cluster [%s] -> %+v", c.String("name"), err)
		return err
	}
	log.Printf("SUCCESS: started cluster [%s]", c.String("name"))
	return nil
}

// ListClusters prints a list of created clusters
func ListClusters(c *cli.Context) error {
	printClusters(c.Bool("all"))
	return nil
}

// GetKubeConfig grabs the kubeconfig from the running cluster and prints the path to stdout
func GetKubeConfig(c *cli.Context) error {
	sourcePath := fmt.Sprintf("%s:/output/kubeconfig.yaml", c.String("name"))
	destPath, _ := getClusterDir(c.String("name"))
	cmd := "docker"
	args := []string{"cp", sourcePath, destPath}
	if err := runCommand(false, cmd, args...); err != nil {
		log.Fatalf("FAILURE: couldn't get kubeconfig for cluster [%s] -> %+v", c.String("name"), err)
		return err
	}
	fmt.Printf("%s\n", path.Join(destPath, "kubeconfig.yaml"))
	return nil
}
