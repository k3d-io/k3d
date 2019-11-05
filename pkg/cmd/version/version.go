package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rancher/k3d/version"
)

func NewCommand() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Args:  cobra.NoArgs,
		Short: "Print k3d and k3s version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("k3d version", version.GetVersion())
			fmt.Println("k3s version", version.GetK3sVersion())
		},
	}
	return versionCmd
}
