/*
Copyright © 2020 The k3d Author(s)

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
	"fmt"
	"regexp"
	"strconv"
	"strings"

	k3d "github.com/rancher/k3d/v4/pkg/types"
	log "github.com/sirupsen/logrus"
)

// Regexp pattern to match node filters
var filterRegexp = regexp.MustCompile(`^(?P<group>server|agent|loadbalancer|all)(?P<subsetSpec>\[(?P<subset>(?P<subsetList>(\d+,?)+)|(?P<subsetRange>\d*:\d*)|(?P<subsetWildcard>\*))\])?$`)

// FilterNodes takes a string filter to return a filtered list of nodes
func FilterNodes(nodes []*k3d.Node, filters []string) ([]*k3d.Node, error) {

	if len(filters) == 0 || len(filters[0]) == 0 {
		log.Warnln("No node filter specified")
		return nodes, nil
	}

	// map roles to subsets
	serverNodes := []*k3d.Node{}
	agentNodes := []*k3d.Node{}
	var serverlb *k3d.Node
	for _, node := range nodes {
		log.Tracef("FilterNodes (%+v): Checking node role %s", filters, node.Role)
		if node.Role == k3d.ServerRole {
			serverNodes = append(serverNodes, node)
		} else if node.Role == k3d.AgentRole {
			agentNodes = append(agentNodes, node)
		} else if node.Role == k3d.LoadBalancerRole {
			serverlb = node
		}
	}

	filteredNodes := []*k3d.Node{}
	set := make(map[*k3d.Node]struct{})

	// range over all instances of group[subset] specs
	for _, filter := range filters {

		// match regex with capturing groups
		match := filterRegexp.FindStringSubmatch(filter)

		if len(match) == 0 {
			return nil, fmt.Errorf("Failed to parse node filters: invalid format or empty subset in '%s'", filter)
		}

		// map capturing group names to submatches
		submatches := MapSubexpNames(filterRegexp.SubexpNames(), match)

		// if one of the filters is 'all', we only return this and drop all others
		if submatches["group"] == "all" {
			if len(filters) > 1 {
				log.Warnf("Node filter 'all' set, but more were specified in '%+v'", filters)
			}
			return nodes, nil
		}

		// Choose the group of nodes to operate on
		groupNodes := []*k3d.Node{}
		if submatches["group"] == string(k3d.ServerRole) {
			groupNodes = serverNodes
		} else if submatches["group"] == string(k3d.AgentRole) {
			groupNodes = agentNodes
		} else if submatches["group"] == string(k3d.LoadBalancerRole) {
			if serverlb == nil {
				return nil, fmt.Errorf("Node filter '%s' targets a node that does not exist (disabled?)", filter)
			}
			filteredNodes = append(filteredNodes, serverlb)
			return filteredNodes, nil // early exit if filtered group is the loadbalancer
		}

		/* Option 1) subset defined by list */
		if submatches["subsetList"] != "" {
			for _, index := range strings.Split(submatches["subsetList"], ",") {
				if index != "" {
					num, err := strconv.Atoi(index)
					if err != nil {
						return nil, fmt.Errorf("Failed to convert subset number to integer in '%s'", filter)
					}
					if num < 0 || num >= len(groupNodes) {
						return nil, fmt.Errorf("Index out of range: index '%d' < 0 or > number of available nodes in filter '%s'", num, filter)
					}
					if _, exists := set[groupNodes[num]]; !exists {
						filteredNodes = append(filteredNodes, groupNodes[num])
						set[groupNodes[num]] = struct{}{}
					}
				}
			}

			/* Option 2) subset defined by range */
		} else if submatches["subsetRange"] != "" {

			/*
			 * subset specified by a range 'START:END', where each side is optional
			 */

			split := strings.Split(submatches["subsetRange"], ":")
			if len(split) != 2 {
				return nil, fmt.Errorf("Failed to parse subset range in '%s'", filter)
			}

			start := 0
			end := len(groupNodes) - 1

			var err error

			if split[0] != "" {
				start, err = strconv.Atoi(split[0])
				if err != nil {
					return nil, fmt.Errorf("Failed to convert subset range start to integer in '%s'", filter)
				}
				if start < 0 || start >= len(groupNodes) {
					return nil, fmt.Errorf("Invalid subset range: start < 0 or > number of available nodes in '%s'", filter)
				}

			}

			if split[1] != "" {
				end, err = strconv.Atoi(split[1])
				if err != nil {
					return nil, fmt.Errorf("Failed to convert subset range start to integer in '%s'", filter)
				}
				if end < start || end >= len(groupNodes) {
					return nil, fmt.Errorf("Invalid subset range: end < start or > number of available nodes in '%s'", filter)
				}
			}

			for i := start; i <= end; i++ {
				if _, exists := set[groupNodes[i]]; !exists {
					filteredNodes = append(filteredNodes, groupNodes[i])
					set[groupNodes[i]] = struct{}{}
				}
			}

			/* Option 3) subset defined by wildcard */
		} else if submatches["subsetWildcard"] == "*" {
			/*
			 * '*' = all nodes
			 */
			for _, node := range groupNodes {
				if _, exists := set[node]; !exists {
					filteredNodes = append(filteredNodes, node)
					set[node] = struct{}{}
				}
			}

			/* Option X) invalid/unknown subset */
		} else {
			return nil, fmt.Errorf("Failed to parse node specifiers: unknown subset in '%s'", filter)
		}

	}

	return filteredNodes, nil
}

// FilterNodesByRole returns a stripped list of nodes which do match the given role
func FilterNodesByRole(nodes []*k3d.Node, role k3d.Role) []*k3d.Node {
	filteredNodes := []*k3d.Node{}
	for _, node := range nodes {
		if node.Role == role {
			filteredNodes = append(filteredNodes, node)
		}
	}
	return filteredNodes
}
