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
package util

import (
	"context"
	"strings"

	k3dcluster "github.com/k3d-io/k3d/v5/pkg/client"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/spf13/cobra"
)

// ValidArgsAvailableClusters is used for shell completion: proposes the list of existing clusters
func ValidArgsAvailableClusters(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var completions []string
	var clusters []*k3d.Cluster
	clusters, err := k3dcluster.ClusterList(context.Background(), runtimes.SelectedRuntime)
	if err != nil {
		l.Log().Errorln("Failed to get list of clusters for shell completion")
		return nil, cobra.ShellCompDirectiveError
	}

clusterLoop:
	for _, cluster := range clusters {
		for _, arg := range args {
			if arg == cluster.Name { // only clusters, that are not in the args yet
				continue clusterLoop
			}
		}
		if strings.HasPrefix(cluster.Name, toComplete) {
			completions = append(completions, cluster.Name)
		}
	}
	return completions, cobra.ShellCompDirectiveDefault
}

// ValidArgsAvailableNodes is used for shell completion: proposes the list of existing nodes
func ValidArgsAvailableNodes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var completions []string
	var nodes []*k3d.Node
	nodes, err := k3dcluster.NodeList(context.Background(), runtimes.SelectedRuntime)
	if err != nil {
		l.Log().Errorln("Failed to get list of nodes for shell completion")
		return nil, cobra.ShellCompDirectiveError
	}

nodeLoop:
	for _, node := range nodes {
		for _, arg := range args {
			if arg == node.Name { // only nodes, that are not in the args yet
				continue nodeLoop
			}
		}
		if strings.HasPrefix(node.Name, toComplete) {
			completions = append(completions, node.Name)
		}
	}
	return completions, cobra.ShellCompDirectiveDefault
}

// ValidArgsAvailableRegistries is used for shell completions: proposes the list of existing registries
func ValidArgsAvailableRegistries(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var completions []string
	var nodes []*k3d.Node
	nodes, err := k3dcluster.NodeList(context.Background(), runtimes.SelectedRuntime)
	if err != nil {
		l.Log().Errorln("Failed to get list of nodes for shell completion")
		return nil, cobra.ShellCompDirectiveError
	}

	nodes = k3dcluster.NodeFilterByRoles(nodes, []k3d.Role{k3d.RegistryRole}, []k3d.Role{})

nodeLoop:
	for _, node := range nodes {
		for _, arg := range args {
			if arg == node.Name { // only nodes, that are not in the args yet
				continue nodeLoop
			}
		}
		if strings.HasPrefix(node.Name, toComplete) {
			completions = append(completions, node.Name)
		}
	}
	return completions, cobra.ShellCompDirectiveDefault
}

// ValidArgsNodeRoles is used for shell completion: proposes the list of possible node roles
func ValidArgsNodeRoles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var completions []string
	roles := []string{string(k3d.ServerRole), string(k3d.AgentRole)}

	for _, role := range roles {
		if strings.HasPrefix(role, toComplete) {
			completions = append(completions, role)
		}
	}
	return completions, cobra.ShellCompDirectiveDefault
}
