package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/urfave/cli"
)

func checkTools(c *cli.Context) error {
	fmt.Println("TEST check")
	return nil
}

func createCluster(c *cli.Context) error {
	fmt.Println("TEST create")
	return nil
}

func deleteCluster(c *cli.Context) error {
	fmt.Println("TEST delete")
	return nil
}

func stopCluster(c *cli.Context) error {
	fmt.Println("TEST stop")
	return nil
}

func startCluster(c *cli.Context) error {
	fmt.Println("TEST start")
	return nil
}

func listClusters(c *cli.Context) error {
	fmt.Println("TEST list")
	return nil
}

func getConfig(c *cli.Context) error {
	fmt.Println("TEST get")
	return nil
}

func main() {

	var clusterName string
	var serverPort int
	var volume string

	app := cli.NewApp()
	app.Name = "k3d"
	app.Usage = "Run k3s in Docker!"

	app.Commands = []cli.Command{
		{
			Name:    "check-tools",
			Aliases: []string{"ct"},
			Usage:   "Check if docker is running",
			Action: func(c *cli.Context) error {
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
					Name:        "name, n",
					Value:       "k3s_default",
					Usage:       "Set a name for the cluster",
					Destination: &clusterName,
				},
				cli.StringFlag{
					Name:        "volume, v",
					Usage:       "Mount a volume into the cluster node (Docker notation: `source:destination`",
					Destination: &volume,
				},
				cli.IntFlag{
					Name:        "port, p",
					Value:       6443,
					Usage:       "Set a port on which the ApiServer will listen",
					Destination: &serverPort,
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
					Name:        "name, n",
					Value:       "k3s_default",
					Usage:       "name of the cluster",
					Destination: &clusterName,
				},
			},
			Action: deleteCluster,
		},
		{
			Name:  "stop",
			Usage: "Stop cluster",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "name, n",
					Value:       "k3s_default",
					Usage:       "name of the cluster",
					Destination: &clusterName,
				},
			},
			Action: func(c *cli.Context) error {
				fmt.Println("Stopping cluster")
				return nil
			},
		},
		{
			Name:  "start",
			Usage: "Start a stopped cluster",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "name, n",
					Value:       "k3s_default",
					Usage:       "name of the cluster",
					Destination: &clusterName,
				},
			},
			Action: func(c *cli.Context) error {
				fmt.Println("Starting stopped cluster")
				return nil
			},
		},
		{
			Name:  "list",
			Usage: "List all clusters",
			Action: func(c *cli.Context) error {
				fmt.Println("Listing clusters")
				return nil
			},
		},
		{
			Name:  "get-config",
			Usage: "Get kubeconfig location for cluster",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "name, n",
					Value:       "k3s_default",
					Usage:       "name of the cluster",
					Destination: &clusterName,
				},
			},
			Action: func(c *cli.Context) error {
				fmt.Println("Starting stopped cluster")
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
