package importimages

import (
	run "github.com/rancher/k3d/cli"

	"github.com/spf13/cobra"

	"github.com/rancher/k3d/pkg/constants"
)

type options struct {
	name     string
	noRemove bool
}

func NewCommand() *cobra.Command {
	// importImagesCmd represents the importImages command
	opts := &options{}
	var importImagesCmd = &cobra.Command{
		Use:     "import-images [image] [image]...",
		Short:   "Import list of container images from your local docker daemon into the cluster",
		Aliases: []string{"import-images", "i"},
		Args:    cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, images []string) error {
			return run.ImportImage(opts.name, opts.noRemove, images)
		},
	}

	importImagesCmd.Flags().StringVarP(&opts.name, "name", "n", constants.DefaultK3sClusterName, "import images into a cluster")
	importImagesCmd.Flags().BoolVarP(&opts.noRemove, "no-remove", "k", false, "Disable automatic removal of the tarball")

	return importImagesCmd
}
