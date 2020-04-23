package main

import (
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
	"github.com/urfave/cli"

	run "github.com/rancher/k3d/cli"
	"github.com/rancher/k3d/version"
)

// defaultK3sImage specifies the default image being used for server and workers
const defaultK3sImage = "docker.io/rancher/k3s"
const defaultK3sClusterName string = "k3s-default"
const defaultRegistryName = "registry.localhost"
const defaultRegistryPort = 5000

// main represents the CLI application
func main() {

	// App Details
	app := cli.NewApp()
	app.Name = "k3d"
	app.Usage = "Run k3s in Docker!"
	app.Version = version.GetVersion()

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
			// shell starts a shell in the context of a running cluster
			Name:  "shell",
			Usage: "Start a subshell for a cluster",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name, n",
					Value: defaultK3sClusterName,
					Usage: "Set a name for the cluster",
				},
				cli.StringFlag{
					Name:  "command, c",
					Usage: "Run a shell command in the context of the cluster",
				},
				cli.StringFlag{
					Name:  "shell, s",
					Value: "auto",
					Usage: "which shell to use. One of [auto, bash, zsh]",
				},
			},
			Action: run.Shell,
		},
		{
			// create creates a new k3s cluster in docker containers
			Name:    "create",
			Aliases: []string{"c"},
			Usage:   "Create a single- or multi-node k3s cluster in docker containers",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name, n",
					Value: defaultK3sClusterName,
					Usage: "Set a name for the cluster",
				},
				cli.StringSliceFlag{
					Name:  "volume, v",
					Usage: "Mount one or more volumes into every node of the cluster (Docker notation: `source:destination`)",
				},
				cli.StringSliceFlag{
					// TODO: remove publish/add-port soon, to clean up
					Name:  "port, p, publish, add-port",
					Usage: "Publish k3s node ports to the host (Format: `-p [ip:][host-port:]container-port[/protocol]@node-specifier`, use multiple options to expose more ports)",
				},
				cli.IntFlag{
					Name:  "port-auto-offset",
					Value: 0,
					Usage: "Automatically add an offset (* worker number) to the chosen host port when using `--publish` to map the same container-port from multiple k3d workers to the host",
				},
				cli.StringFlag{
					Name:  "api-port, a",
					Value: "6443",
					Usage: "Specify the Kubernetes cluster API server port (Format: `-a [host:]port`",
				},
				cli.IntFlag{
					Name:  "wait, t",
					Value: -1,
					Usage: "Wait for a maximum of `TIMEOUT` seconds (>= 0) for the cluster to be ready and rollback if it doesn't come up in time. Disabled by default (-1).",
				},
				cli.StringFlag{
					Name:  "image, i",
					Usage: "Specify a k3s image (Format: <repo>/<image>:<tag>)",
					Value: fmt.Sprintf("%s:%s", defaultK3sImage, version.GetK3sVersion()),
				},
				cli.StringSliceFlag{
					Name:  "server-arg, x",
					Usage: "Pass an additional argument to k3s server (new flag per argument)",
				},
				cli.StringSliceFlag{
					Name:  "agent-arg",
					Usage: "Pass an additional argument to k3s agent (new flag per argument)",
				},
				cli.StringSliceFlag{
					Name:  "env, e",
					Usage: "Pass an additional environment variable (new flag per variable)",
				},
				cli.StringSliceFlag{
					Name:  "label, l",
					Usage: "Add a docker label to node container (Format: `key[=value][@node-specifier]`, new flag per label)",
				},
				cli.IntFlag{
					Name:  "workers, w",
					Value: 0,
					Usage: "Specify how many worker nodes you want to spawn",
				},
				cli.BoolFlag{
					Name:  "auto-restart",
					Usage: "Set docker's --restart=unless-stopped flag on the containers",
				},
				cli.BoolFlag{
					Name:  "enable-registry",
					Usage: "Start a local Docker registry",
				},
				cli.StringFlag{
					Name:  "registry-name",
					Value: defaultRegistryName,
					Usage: "Name of the local registry container",
				},
				cli.IntFlag{
					Name:  "registry-port",
					Value: defaultRegistryPort,
					Usage: "Port of the local registry container",
				},
				cli.StringFlag{
					Name:  "registry-volume",
					Usage: "Use a specific volume for the registry storage (will be created if not existing)",
				},
				cli.StringFlag{
					Name:  "registries-file",
					Usage: "registries.yaml config file",
				},
				cli.BoolFlag{
					Name:  "enable-registry-cache",
					Usage: "Use the local registry as a cache for the Docker Hub (Note: This disables pushing local images to the registry!)",
				},
			},
			Action: run.CreateCluster,
		},
		/*
		 * Add a new node to an existing k3d/k3s cluster (choosing k3d by default)
		 */
		{
			Name:  "add-node",
			Usage: "[EXPERIMENTAL] Add nodes to an existing k3d/k3s cluster (k3d by default)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "role, r",
					Usage: "Choose role of the node you want to add [agent|server]",
					Value: "agent",
				},
				cli.StringFlag{
					Name:  "name, n",
					Usage: "Name of the k3d cluster that you want to add a node to [only for node name if --k3s is set]",
					Value: defaultK3sClusterName,
				},
				cli.IntFlag{
					Name:  "count, c",
					Usage: "Number of nodes that you want to add",
					Value: 1,
				},
				cli.StringFlag{
					Name:  "image, i",
					Usage: "Specify a k3s image (Format: <repo>/<image>:<tag>)",
					Value: fmt.Sprintf("%s:%s", defaultK3sImage, version.GetK3sVersion()),
				},
				cli.StringSliceFlag{
					Name:  "arg, x",
					Usage: "Pass arguments to the k3s server/agent command.",
				},
				cli.StringSliceFlag{
					Name:  "env, e",
					Usage: "Pass an additional environment variable (new flag per variable)",
				},
				cli.StringSliceFlag{
					Name:  "volume, v",
					Usage: "Mount one or more volumes into every created node (Docker notation: `source:destination`)",
				},
				/*
				 * Connect to a non-dockerized k3s cluster
				 */
				cli.StringFlag{
					Name:  "k3s",
					Usage: "Add a k3d node to a non-k3d k3s cluster (specify k3s server URL like this `https://<host>:<port>`) [requires k3s-secret or k3s-token]",
				},
				cli.StringFlag{
					Name:  "k3s-secret, s",
					Usage: "Specify k3s cluster secret (or use --k3s-token to use a node token)",
				},
				cli.StringFlag{
					Name:  "k3s-token, t",
					Usage: "Specify k3s node token (or use --k3s-secret to use a cluster secret)[overrides k3s-secret]",
				},
			},
			Action: run.AddNode,
		},
		{
			// delete deletes an existing k3s cluster (remove container and cluster directory)
			Name:    "delete",
			Aliases: []string{"d", "del"},
			Usage:   "Delete cluster",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name, n",
					Value: defaultK3sClusterName,
					Usage: "name of the cluster",
				},
				cli.BoolFlag{
					Name:  "all, a",
					Usage: "Delete all existing clusters (this ignores the --name/-n flag)",
				},
				cli.BoolFlag{
					Name:  "prune",
					Usage: "Disconnect any other non-k3d containers in the network before deleting the cluster",
				},
				cli.BoolFlag{
					Name:  "keep-registry-volume",
					Usage: "Do not delete the registry volume",
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
					Value: defaultK3sClusterName,
					Usage: "Name of the cluster",
				},
				cli.BoolFlag{
					Name:  "all, a",
					Usage: "Stop all running clusters (this ignores the --name/-n flag)",
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
					Value: defaultK3sClusterName,
					Usage: "Name of the cluster",
				},
				cli.BoolFlag{
					Name:  "all, a",
					Usage: "Start all stopped clusters (this ignores the --name/-n flag)",
				},
			},
			Action: run.StartCluster,
		},
		{
			// list prints a list of created clusters
			Name:    "list",
			Aliases: []string{"ls", "l"},
			Usage:   "List all clusters",
			Action:  run.ListClusters,
		},
		{
			// get-kubeconfig grabs the kubeconfig from the cluster and prints the path to it
			Name:  "get-kubeconfig",
			Usage: "Get kubeconfig location for cluster",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name, n",
					Value: defaultK3sClusterName,
					Usage: "Name of the cluster",
				},
				cli.BoolFlag{
					Name:  "all, a",
					Usage: "Get kubeconfig for all clusters (this ignores the --name/-n flag)",
				},
				cli.BoolFlag{
					Name:  "overwrite, o",
					Usage: "Overwrite any existing file with the same name",
				},
			},
			Action: run.GetKubeConfig,
		},
		{
			// get-kubeconfig grabs the kubeconfig from the cluster and prints the path to it
			Name:    "import-images",
			Aliases: []string{"i"},
			Usage:   "Import a comma- or space-separated list of container images from your local docker daemon into the cluster",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name, n, cluster, c",
					Value: defaultK3sClusterName,
					Usage: "Name of the cluster",
				},
				cli.BoolFlag{
					Name:  "no-remove, no-rm, keep, k",
					Usage: "Disable automatic removal of the tarball",
				},
			},
			Action: run.ImportImage,
		},
		{
			Name:  "version",
			Usage: "print k3d and k3s version",
			Action: func(c *cli.Context) {
				fmt.Println("k3d version", version.GetVersion())
				fmt.Println("k3s version", version.GetK3sVersion())
			},
		},
	}

	// Global flags
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "Enable verbose output",
		},
		cli.BoolFlag{
			Name:  "timestamp",
			Usage: "Enable timestamps in logs messages",
		},
	}

	// init log level
	app.Before = func(c *cli.Context) error {
		log.SetOutput(ioutil.Discard)
		log.AddHook(&writer.Hook{
			Writer: os.Stderr,
			LogLevels: []log.Level{
				log.PanicLevel,
				log.FatalLevel,
				log.ErrorLevel,
				log.WarnLevel,
			},
		})
		log.AddHook(&writer.Hook{
			Writer: os.Stdout,
			LogLevels: []log.Level{
				log.InfoLevel,
				log.DebugLevel,
			},
		})
		log.SetFormatter(&log.TextFormatter{
			ForceColors: true,
		})
		if c.GlobalBool("verbose") {
			log.SetLevel(log.DebugLevel)
		} else {
			log.SetLevel(log.InfoLevel)
		}
		if c.GlobalBool("timestamp") {
			log.SetFormatter(&log.TextFormatter{
				FullTimestamp: true,
				ForceColors:   true,
			})
		}

		return nil
	}

	// run the whole thing
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
