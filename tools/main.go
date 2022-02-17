package main

import (
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	run "github.com/k3d-io/k3d/tools/cmd"
	"github.com/k3d-io/k3d/tools/version"
	"github.com/urfave/cli"
)

// main represents the CLI application
func main() {

	// App Details
	app := cli.NewApp()
	app.Name = "k3d-tools"
	app.Usage = "Tools to help running k3d successfully!"
	app.Version = version.GetVersion()

	// commands that you can execute
	app.Commands = []cli.Command{
		{
			// save-image
			Name:    "save-image",
			Aliases: []string{"save"},
			Usage:   "Save images to tarball",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "destination, dest, d",
					Value: "/images",
					Usage: "destination tar-file (optional)",
				},
				cli.StringFlag{
					Name:  "cluster, c",
					Value: "k3s-default",
					Usage: "name of the k3d cluster",
				},
			},
			Action: run.ImageSave,
		},
		{
			Name:  "noop",
			Usage: "Don't do anything and sleep forever",
			Action: func(c *cli.Context) {
				for {
					fmt.Println("Sleeping for 12h")
					time.Sleep(12 * time.Hour)
				}
			},
		},
	}

	// run the whole thing
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
