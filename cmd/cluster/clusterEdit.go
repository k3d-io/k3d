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
package cluster

import (
	cliutil "github.com/k3d-io/k3d/v5/cmd/util"
	"github.com/k3d-io/k3d/v5/pkg/client"
	conf "github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/spf13/cobra"
)

// NewCmdClusterEdit returns a new cobra command
func NewCmdClusterEdit() *cobra.Command {
	// create new cobra command
	cmd := &cobra.Command{
		Use:               "edit CLUSTER",
		Short:             "[EXPERIMENTAL] Edit cluster(s).",
		Long:              `[EXPERIMENTAL] Edit cluster(s).`,
		Args:              cobra.ExactArgs(1),
		Aliases:           []string{"update"},
		ValidArgsFunction: cliutil.ValidArgsAvailableClusters,
		Run: func(cmd *cobra.Command, args []string) {
			existingCluster, changeset := parseEditClusterCmd(cmd, args)

			l.Log().Debugf("===== Current =====\n%+v\n===== Changeset =====\n%+v\n", existingCluster, changeset)

			if err := client.ClusterEditChangesetSimple(cmd.Context(), runtimes.SelectedRuntime, existingCluster, changeset); err != nil {
				l.Log().Fatalf("Failed to update the cluster: %v", err)
			}

			l.Log().Infof("Successfully updated %s", existingCluster.Name)
		},
	}

	// add subcommands

	// add flags
	cmd.Flags().StringArray("port-add", nil, "Map ports from the node containers (via the serverlb) to the host (Format: `[HOST:][HOSTPORT:]CONTAINERPORT[/PROTOCOL][@NODEFILTER]`)\n - Example: `k3d cluster edit k3d-mycluster-serverlb --port-add 8080:80`")
	cmd.Flags().StringArray("port-delete", nil, "[EXPERIMENTAL] Delete a port mapping with the given format\nThe mapping spec needs to be exactly the same as the one used during creation\n - Example: `k3d cluster edit k3d-mycluster-serverlb --port-delete 8080:80`")

	// done
	return cmd
}

// parseEditClusterCmd parses the command input into variables required to delete nodes
func parseEditClusterCmd(cmd *cobra.Command, args []string) (*k3d.Cluster, *conf.SimpleConfig) {
	existingCluster, err := client.ClusterGet(cmd.Context(), runtimes.SelectedRuntime, &k3d.Cluster{Name: args[0]})
	if err != nil {
		l.Log().Fatalln(err)
	}

	if existingCluster == nil {
		l.Log().Infof("Cluster %s not found", args[0])
		return nil, nil
	}

	changeset := conf.SimpleConfig{}
	changeset.Ports = []conf.PortWithNodeFilters{}

	portsAdded := parsePortChangeFlag(cmd, &changeset, "port-add", false)

	portsDeleted := parsePortChangeFlag(cmd, &changeset, "port-delete", true)

	if portsAdded && portsDeleted {
		l.Log().Fatalln("Cannot combine port addition and deletion")
	}

	return existingCluster, &changeset
}

func parsePortChangeFlag(cmd *cobra.Command, changeset *conf.SimpleConfig, flagName string, isRemoval bool) bool {
	portFlags, err := cmd.Flags().GetStringArray(flagName)
	if err != nil {
		l.Log().Fatalln(err)
	}

	portFilterMap := make(map[string][]string, 1)
	for _, portFlag := range portFlags {
		// split node filter from the specified volume
		portmap, filters, err := cliutil.SplitFiltersFromFlag(portFlag)
		if err != nil {
			l.Log().Fatalln(err)
		}

		// create new entry or append filter to existing entry
		if _, exists := portFilterMap[portmap]; exists {
			l.Log().Fatalln("Same Portmapping can not be used for multiple nodes")
		} else {
			portFilterMap[portmap] = filters
		}
	}
	l.Log().Tracef("PortFilterMap: %+v", portFilterMap)

	for port, nodeFilters := range portFilterMap {
		changeset.Ports = append(changeset.Ports, conf.PortWithNodeFilters{
			Port:        port,
			NodeFilters: nodeFilters,
			Removal:     isRemoval,
		})
	}
	return len(portFilterMap) > 0
}
