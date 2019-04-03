package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/urfave/cli"
)

func createCluster(c *cli.Context) error {
	createClusterDir(c.String("name"))
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
		for _, extra := range extraArgs {
			args = append(args, extra)
		}
	}
	args = append(args,
		"-d",
		image,
		"server",                                // cmd
		"--https-listen-port", c.String("port"), //args
	)
	log.Printf("Creating cluster [%s]", c.String("name"))
	log.Printf("Running command: %+v", exec.Command(cmd, args...).Args)
	if err := exec.Command(cmd, args...).Run(); err != nil {
		log.Fatalf("FAILURE: couldn't create cluster [%s] -> %+v", c.String("name"), err)
		return err
	}
	log.Printf("SUCCESS: created cluster [%s]", c.String("name"))
	return nil
}

func deleteCluster(c *cli.Context) error {
	cmd := "docker"
	args := []string{"rm", c.String("name")}
	log.Printf("Deleting cluster [%s]", c.String("name"))
	log.Printf("Running command: %+v", exec.Command(cmd, args...).Args)
	if err := exec.Command(cmd, args...).Run(); err != nil {
		log.Printf("WARNING: couldn't delete cluster [%s], trying a force remove now.", c.String("name"))
		args = append(args, "-f")
		log.Printf("Running command: %+v", exec.Command(cmd, args...).Args)
		if err := exec.Command(cmd, args...).Run(); err != nil {
			log.Fatalf("FAILURE: couldn't delete cluster [%s] -> %+v", c.String("name"), err)
			return err
		}
	}
	deleteClusterDir(c.String("name"))
	log.Printf("SUCCESS: deleted cluster [%s]", c.String("name"))
	return nil
}

func stopCluster(c *cli.Context) error {
	cmd := "docker"
	args := []string{"stop", c.String("name")}
	log.Printf("Stopping cluster [%s]", c.String("name"))
	log.Printf("Running command: %+v", exec.Command(cmd, args...).Args)
	if err := exec.Command(cmd, args...).Run(); err != nil {
		log.Fatalf("FAILURE: couldn't stop cluster [%s] -> %+v", c.String("name"), err)
		return err
	}
	log.Printf("SUCCESS: stopped cluster [%s]", c.String("name"))
	return nil
}

func startCluster(c *cli.Context) error {
	cmd := "docker"
	args := []string{"start", c.String("name")}
	log.Printf("Starting cluster [%s]", c.String("name"))
	log.Printf("Running command: %+v", exec.Command(cmd, args...).Args)
	if err := exec.Command(cmd, args...).Run(); err != nil {
		log.Fatalf("FAILURE: couldn't start cluster [%s] -> %+v", c.String("name"), err)
		return err
	}
	log.Printf("SUCCESS: started cluster [%s]", c.String("name"))
	return nil
}

func listClusters(c *cli.Context) error {
	listClusterDirs()
	return nil
}

func getKubeConfig(c *cli.Context) error {
	sourcePath := fmt.Sprintf("%s:/output/kubeconfig.yaml", c.String("name"))
	destPath, _ := getClusterDir(c.String("name"))
	cmd := "docker"
	args := []string{"cp", sourcePath, destPath}
	log.Printf("Grabbing kubeconfig for cluster [%s]", c.String("name"))
	log.Printf("Running command: %+v", exec.Command(cmd, args...).Args)
	if err := exec.Command(cmd, args...).Run(); err != nil {
		log.Fatalf("FAILURE: couldn't get kubeconfig for cluster [%s] -> %+v", c.String("name"), err)
		return err
	}
	log.Printf("SUCCESS: retrieved kubeconfig for cluster [%s]", c.String("name"))
	return nil
}

func main() {

	app := cli.NewApp()
	app.Name = "k3d"
	app.Usage = "Run k3s in Docker!"
	app.Version = "0.0.1"
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "iwilltry42",
			Email: "iwilltry42@gmail.com",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "check-tools",
			Aliases: []string{"ct"},
			Usage:   "Check if docker is running",
			Action: func(c *cli.Context) error {
				//TODO: own function with version check
				log.Print("Checking docker...")
				cmd := "docker"
				args := []string{"version"}
				if err := exec.Command(cmd, args...).Run(); err != nil {
					log.Fatalf("Checking docker: FAILED")
					return err
				}
				log.Println("Checking docker: SUCCESS")
				return nil
			},
		},
		{
			Name:    "create",
			Aliases: []string{"c"},
			Usage:   "Create a single node k3s cluster in a container",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name, n",
					Value: "k3s_default",
					Usage: "Set a name for the cluster",
				},
				cli.StringFlag{
					Name:  "volume, v",
					Usage: "Mount a volume into the cluster node (Docker notation: `source:destination`",
				},
				cli.StringFlag{
					Name:  "version",
					Value: "v0.1.0",
					Usage: "Choose the k3s image version",
				},
				cli.IntFlag{
					Name:  "port, p",
					Value: 6443,
					Usage: "Set a port on which the ApiServer will listen",
				},
			},
			Action: createCluster,
		},
		{
			Name:    "delete",
			Aliases: []string{"d"},
			Usage:   "Delete cluster",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name, n",
					Value: "k3s_default",
					Usage: "name of the cluster",
				},
			},
			Action: deleteCluster,
		},
		{
			Name:  "stop",
			Usage: "Stop cluster",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name, n",
					Value: "k3s_default",
					Usage: "name of the cluster",
				},
			},
			Action: stopCluster,
		},
		{
			Name:  "start",
			Usage: "Start a stopped cluster",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name, n",
					Value: "k3s_default",
					Usage: "name of the cluster",
				},
			},
			Action: startCluster,
		},
		{
			Name:   "list",
			Usage:  "List all clusters",
			Action: listClusters,
		},
		{
			Name:  "get-kubeconfig",
			Usage: "Get kubeconfig location for cluster",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name, n",
					Value: "k3s_default",
					Usage: "name of the cluster",
				},
			},
			Action: getKubeConfig,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
