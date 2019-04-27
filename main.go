package main

import (
	"log"
	"os"

	run "github.com/rancher/k3d/cli"
	"github.com/rancher/k3d/version"
	"github.com/urfave/cli"
)

// main represents the CLI application
func main() {

	// App Details
	app := cli.NewApp()
	app.Name = "k3d"
	app.Usage = "Run k3s in Docker!"
	app.Version = version.GetVersion()
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Thorsten Klein",
			Email: "iwilltry42@gmail.com",
		},
		cli.Author{
			Name:  "Rishabh Gupta",
			Email: "r.g.gupta@outlook.com",
		},
		cli.Author{
			Name: "Darren Shepherd",
		},
	}

	// commands that you can execute
	app.Commands = []cli.Command{
		{
			// check-tools verifies that docker is up and running
			Name:    "check-tools",
			Aliases: []string{"ct"},
			Usage:   "Check if docker is running",
			Action:  run.CheckTools,
		},
		{
			// create creates a new k3s cluster in a container
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
					Usage: "Mount one or more volumes into the cluster node (Docker notation: `source:destination[,source:destination]`)",
				},
				cli.StringFlag{
					Name:  "version",
					Value: version.GetK3sVersion(),
					Usage: "Choose the k3s image version",
				},
				cli.IntFlag{
					Name:  "port, p",
					Value: 6443,
					Usage: "Set a port on which the ApiServer will listen",
				},
				cli.IntFlag{
					Name:  "timeout, t",
					Value: 0,
					Usage: "Set the timeout value when --wait flag is set",
				},
				cli.BoolFlag{
					Name:  "wait, w",
					Usage: "Wait for the cluster to come up",
				},
				cli.StringSliceFlag{
					Name:  "server-arg, x",
					Usage: "Pass an additional argument to k3s server (new flag per argument)",
				},
				cli.StringSliceFlag{
					Name:  "env, e",
					Usage: "Pass an additional environment variable (new flag per variable)",
				},
				cli.IntFlag{
					Name:  "workers",
					Value: 0,
					Usage: "Specify how many worker nodes you want to spawn",
				},
			},
			Action: run.CreateCluster,
		},
		{
			// delete deletes an existing k3s cluster (remove container and cluster directory)
			Name:    "delete",
			Aliases: []string{"d", "del"},
			Usage:   "Delete cluster",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name, n",
					Value: "k3s_default",
					Usage: "name of the cluster",
				},
				cli.BoolFlag{
					Name:  "all, a",
					Usage: "delete all existing clusters (this ignores the --name/-n flag)",
				},
			},
			Action: run.DeleteCluster,
		},
		{
			// stop stopy a running cluster (its container) so it's restartable
			Name:  "stop",
			Usage: "Stop cluster",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name, n",
					Value: "k3s_default",
					Usage: "name of the cluster",
				},
				cli.BoolFlag{
					Name:  "all, a",
					Usage: "stop all running clusters (this ignores the --name/-n flag)",
				},
			},
			Action: run.StopCluster,
		},
		{
			// start restarts a stopped cluster container
			Name:  "start",
			Usage: "Start a stopped cluster",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name, n",
					Value: "k3s_default",
					Usage: "name of the cluster",
				},
				cli.BoolFlag{
					Name:  "all, a",
					Usage: "start all stopped clusters (this ignores the --name/-n flag)",
				},
			},
			Action: run.StartCluster,
		},
		{
			// list prints a list of created clusters
			Name:    "list",
			Aliases: []string{"ls", "l"},
			Usage:   "List all clusters",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "all, a",
					Usage: "also show non-running clusters",
				},
			},
			Action: run.ListClusters,
		},
		{
			// get-kubeconfig grabs the kubeconfig from the cluster and prints the path to it
			Name:  "get-kubeconfig",
			Usage: "Get kubeconfig location for cluster",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name, n",
					Value: "k3s_default",
					Usage: "name of the cluster",
				},
				cli.BoolFlag{
					Name:  "all, a",
					Usage: "get kubeconfig for all clusters (this ignores the --name/-n flag)",
				},
			},
			Action: run.GetKubeConfig,
		},
	}

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "Enable verbose output",
		},
	}

	// run the whole thing
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
