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
package client

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"

	"github.com/k3d-io/k3d/v5/pkg/config/types"
	config "github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/pkg/util"
)

var (
	ErrNodeAddPortsExists error = errors.New("port exists on target")
)

func TransformPorts(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster, portsWithNodeFilters []config.PortWithNodeFilters) error {
	nodeCount := len(cluster.Nodes)
	nodeList := cluster.Nodes

	for _, portWithNodeFilters := range portsWithNodeFilters {
		l.Log().Tracef("inspecting port mapping for %s with nodefilters %s", portWithNodeFilters.Port, portWithNodeFilters.NodeFilters)
		if len(portWithNodeFilters.NodeFilters) == 0 && nodeCount > 1 {
			l.Log().Infof("portmapping '%s' lacks a nodefilter, but there's more than one node: defaulting to %s", portWithNodeFilters.Port, types.DefaultTargetsNodefiltersPortMappings)
			portWithNodeFilters.NodeFilters = types.DefaultTargetsNodefiltersPortMappings
		}

		for _, f := range portWithNodeFilters.NodeFilters {
			if strings.HasPrefix(f, "loadbalancer") {
				l.Log().Infof("portmapping '%s' targets the loadbalancer: defaulting to %s", portWithNodeFilters.Port, types.DefaultTargetsNodefiltersPortMappings)
				portWithNodeFilters.NodeFilters = types.DefaultTargetsNodefiltersPortMappings
				break
			}
		}

		filteredNodes, err := util.FilterNodesWithSuffix(nodeList, portWithNodeFilters.NodeFilters, "proxy", "direct") // TODO: move "proxy" and "direct" allowed suffices to constants
		if err != nil {
			return err
		}

		for suffix, nodes := range filteredNodes {
			// skip, if no nodes in filtered set, so we don't add portmappings with no targets in the backend
			if len(nodes) == 0 {
				continue
			}
			portmappings, err := nat.ParsePortSpec(portWithNodeFilters.Port)
			if err != nil {
				return fmt.Errorf("error parsing port spec '%s': %+v", portWithNodeFilters.Port, err)
			}

			if suffix == "proxy" || suffix == util.NodeFilterSuffixNone { // proxy is the default suffix for port mappings
				if cluster.ServerLoadBalancer == nil {
					return fmt.Errorf("port-mapping of type 'proxy' specified, but loadbalancer is disabled")
				}
				if err := addPortMappings(cluster.ServerLoadBalancer.Node, portmappings); err != nil {
					return err
				}
				for _, pm := range portmappings {
					if err := loadbalancerAddPortConfigs(cluster.ServerLoadBalancer, pm, nodes); err != nil {
						return fmt.Errorf("error adding port config to loadbalancer: %w", err)
					}
				}
			} else if suffix == "direct" {
				if len(nodes) > 1 {
					return fmt.Errorf("error: cannot apply a direct port-mapping (%s) to more than one node", portmappings)
				}
				for _, node := range nodes {
					if err := addPortMappings(node, portmappings); err != nil {
						return err
					}
				}
			} else if suffix != util.NodeFilterMapKeyAll {
				return fmt.Errorf("error adding port mappings: unknown suffix %s", suffix)
			}
		}
	}

	// print generated loadbalancer config if exists
	// (avoid segmentation fault if loadbalancer is disabled)
	if l.Log().GetLevel() >= logrus.DebugLevel && cluster.ServerLoadBalancer != nil {
		yamlized, err := yaml.Marshal(cluster.ServerLoadBalancer.Config)
		if err != nil {
			l.Log().Errorf("error printing loadbalancer config: %v", err)
		} else {
			l.Log().Debugf("generated loadbalancer config:\n%s", string(yamlized))
		}
	}
	return nil
}

func addPortMappings(node *k3d.Node, portmappings []nat.PortMapping) error {
	if node.Ports == nil {
		node.Ports = nat.PortMap{}
	}
	for _, pm := range portmappings {
		node.Ports[pm.Port] = append(node.Ports[pm.Port], pm.Binding)
	}
	return nil
}
