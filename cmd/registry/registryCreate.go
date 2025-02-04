/*
Copyright © 2020-2023 The k3d Author(s)

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package registry

import (
	"fmt"

	l "github.com/k3d-io/k3d/v5/pkg/logger"

	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"

	"github.com/k3d-io/k3d/v5/pkg/client"

	cliutil "github.com/k3d-io/k3d/v5/cmd/util"
	"github.com/spf13/cobra"
)

type regCreatePreProcessedFlags struct {
	Port     string
	Clusters []string
	Volumes  []string
}

type regCreateFlags struct {
	Image          string
	Network        string
	ProxyRemoteURL string
	ProxyUsername  string
	ProxyPassword  string
	NoHelp         bool
	DeleteEnabled  bool
}

var helptext string = `# You can now use the registry like this (example):
# 1. create a new cluster that uses this registry
k3d cluster create --registry-use %s

# 2. tag an existing local image to be pushed to the registry
docker tag nginx:latest %s/mynginx:v0.1

# 3. push that image to the registry
docker push %s/mynginx:v0.1

# 4. run a pod that uses this image
kubectl run mynginx --image %s/mynginx:v0.1
`

// NewCmdRegistryCreate returns a new cobra command
func NewCmdRegistryCreate() *cobra.Command {
	flags := &regCreateFlags{}
	ppFlags := &regCreatePreProcessedFlags{}

	// create new command
	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a new registry",
		Long:  `Create a new registry.`,
		Args:  cobra.MaximumNArgs(1), // maximum one name accepted
		Run: func(cmd *cobra.Command, args []string) {
			reg, clusters := parseCreateRegistryCmd(cmd, args, flags, ppFlags)
			regNode, err := client.RegistryRun(cmd.Context(), runtimes.SelectedRuntime, reg)
			if err != nil {
				l.Log().Fatalln(err)
			}
			if err := client.RegistryConnectClusters(cmd.Context(), runtimes.SelectedRuntime, regNode, clusters); err != nil {
				l.Log().Errorln(err)
			}
			l.Log().Infof("Successfully created registry '%s'", reg.Host)
			regString := fmt.Sprintf("%s:%s", reg.Host, reg.ExposureOpts.Binding.HostPort)
			if !flags.NoHelp {
				fmt.Println(fmt.Sprintf(helptext, regString, regString, regString, regString))
			}
		},
	}

	// add flags

	// TODO: connecting to clusters requires non-existing config reload functionality in containerd
	cmd.Flags().StringArrayVarP(&ppFlags.Clusters, "cluster", "c", nil, "[NotReady] Select the cluster(s) that the registry shall connect to.")
	if err := cmd.RegisterFlagCompletionFunc("cluster", cliutil.ValidArgsAvailableClusters); err != nil {
		l.Log().Fatalln("Failed to register flag completion for '--cluster'", err)
	}
	if err := cmd.Flags().MarkHidden("cluster"); err != nil {
		l.Log().Fatalln("Failed to hide --cluster flag on registry create command")
	}

	cmd.Flags().StringVarP(&flags.Image, "image", "i", fmt.Sprintf("%s:%s", k3d.DefaultRegistryImageRepo, k3d.DefaultRegistryImageTag), "Specify image used for the registry")

	cmd.Flags().StringVarP(&ppFlags.Port, "port", "p", "random", "Select which port the registry should be listening on on your machine (localhost) (Format: `[HOST:]HOSTPORT`)\n - Example: `k3d registry create --port 0.0.0.0:5111`")
	cmd.Flags().StringArrayVarP(&ppFlags.Volumes, "volume", "v", nil, "Mount volumes into the registry node (Format: `[SOURCE:]DEST`")

	cmd.Flags().StringVar(&flags.Network, "default-network", k3d.DefaultRuntimeNetwork, "Specify the network connected to the registry")
	cmd.Flags().StringVar(&flags.ProxyRemoteURL, "proxy-remote-url", "", "Specify the url of the proxied remote registry")
	cmd.Flags().StringVar(&flags.ProxyUsername, "proxy-username", "", "Specify the username of the proxied remote registry")
	cmd.Flags().StringVar(&flags.ProxyPassword, "proxy-password", "", "Specify the password of the proxied remote registry")

	cmd.Flags().BoolVar(&flags.NoHelp, "no-help", false, "Disable the help text (How-To use the registry)")
	cmd.Flags().BoolVar(&flags.DeleteEnabled, "delete-enabled", false, "Enable image deletion")

	// done
	return cmd
}

// parseCreateRegistryCmd parses the command input into variables required to create a registry
func parseCreateRegistryCmd(cmd *cobra.Command, args []string, flags *regCreateFlags, ppFlags *regCreatePreProcessedFlags) (*k3d.Registry, []*k3d.Cluster) {
	// --cluster
	clusters := []*k3d.Cluster{}
	for _, name := range ppFlags.Clusters {
		clusters = append(clusters,
			&k3d.Cluster{
				Name: name,
			},
		)
	}

	// --port
	exposePort, err := cliutil.ParseRegistryPortExposureSpec(ppFlags.Port)
	if err != nil {
		l.Log().Errorln("Failed to parse registry port")
		l.Log().Fatalln(err)
	}

	// set the name for the registry node
	registryName := ""
	if len(args) > 0 {
		registryName = fmt.Sprintf("%s-%s", k3d.DefaultObjectNamePrefix, args[0])
	}

	// -- proxy
	var options k3d.RegistryOptions

	if flags.ProxyRemoteURL != "" {
		proxy := k3d.RegistryProxy{
			RemoteURL: flags.ProxyRemoteURL,
			Username:  flags.ProxyUsername,
			Password:  flags.ProxyPassword,
		}
		options.Proxy = proxy
		l.Log().Traceln("Proxy info:", proxy)
	}

	// --volume
	var volumes []string
	if len(ppFlags.Volumes) > 0 {
		volumes = []string{}

		for _, volumeFlag := range ppFlags.Volumes {
			volume, _, err := cliutil.SplitFiltersFromFlag(volumeFlag)
			if err != nil {
				l.Log().Fatalln(err)
			}
			volumes = append(volumes, volume)
		}
	}

	options.DeleteEnabled = flags.DeleteEnabled

	return &k3d.Registry{Host: registryName, Image: flags.Image, ExposureOpts: *exposePort, Network: flags.Network, Options: options, Volumes: volumes}, clusters
}
