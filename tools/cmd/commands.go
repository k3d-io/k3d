package run

import "github.com/urfave/cli"

func ImageSave(c *cli.Context) error {
	return imageSave(c.Args(), c.String("destination"), c.String("cluster"))
}
