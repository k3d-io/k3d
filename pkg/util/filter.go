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
	"fmt"
	"regexp"
	"strconv"
	"strings"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

const (
	NodeFilterSuffixNone = "nosuffix"
	NodeFilterMapKeyAll  = "all"
)

var (
	rolesByIdentifier = map[string]k3d.Role{
		"server":       k3d.ServerRole,
		"servers":      k3d.ServerRole,
		"agent":        k3d.AgentRole,
		"agents":       k3d.AgentRole,
		"loadbalancer": k3d.LoadBalancerRole,
	}
)

// Regexp pattern to match node filters
var NodeFilterRegexp = regexp.MustCompile(`^(?P<group>server|servers|agent|agents|loadbalancer|all)(?P<subsetSpec>:(?P<subset>(?P<subsetList>(\d+,?)+)|(?P<subsetRange>\d*-\d*)|(?P<subsetWildcard>\*)))?(?P<suffixSpec>:(?P<suffix>[[:alpha:]]+))?$`)

// FilterNodesBySuffix properly interprets NodeFilters with suffix
func FilterNodesWithSuffix(nodes []*k3d.Node, nodefilters []string, allowedSuffices ...string) (map[string][]*k3d.Node, error) {
	if len(nodefilters) == 0 || len(nodefilters[0]) == 0 {
		return nil, fmt.Errorf("No nodefilters specified")
	}

	result := map[string][]*k3d.Node{
		NodeFilterMapKeyAll:  nodes,
		NodeFilterSuffixNone: make([]*k3d.Node, 0),
	}
	for _, s := range allowedSuffices {
		result[s] = make([]*k3d.Node, 0) // init map for this suffix, if not exists
	}

	for _, nf := range nodefilters {
		suffix := NodeFilterSuffixNone

		// match regex with capturing groups
		match := NodeFilterRegexp.FindStringSubmatch(nf)

		if len(match) == 0 {
			return nil, fmt.Errorf("Failed to parse node filters (with suffix): invalid format or empty subset in '%s'", nf)
		}

		// map capturing group names to submatches
		submatches := MapSubexpNames(NodeFilterRegexp.SubexpNames(), match)

		// get suffix
		cleanedNf := nf
		if sf, ok := submatches["suffix"]; ok && sf != "" {
			suffix = sf
			cleanedNf = strings.TrimSuffix(nf, submatches["suffixSpec"])
		}

		// suffix not in result map, meaning, that it's also not allowed
		if _, ok := result[suffix]; !ok {
			return nil, fmt.Errorf("error filtering nodes: unallowed suffix '%s' in nodefilter '%s'", suffix, nf)
		}

		filteredNodes, err := FilterNodes(nodes, []string{cleanedNf})
		if err != nil {
			return nil, fmt.Errorf("failed to filter nodes by filter '%s': %w", nf, err)
		}

		l.Log().Tracef("Filtered %d nodes for suffix '%s' (filter: %s)", len(filteredNodes), suffix, nf)

		result[suffix] = append(result[suffix], filteredNodes...)
	}

	return result, nil
}

// FilterNodes takes a string filter to return a filtered list of nodes
func FilterNodes(nodes []*k3d.Node, filters []string) ([]*k3d.Node, error) {
	l.Log().Tracef("Filtering %d nodes by %s", len(nodes), filters)

	if len(filters) == 0 || len(filters[0]) == 0 {
		l.Log().Warnln("No node filter specified")
		return nodes, nil
	}

	// map roles to subsets
	serverNodes := []*k3d.Node{}
	agentNodes := []*k3d.Node{}
	var serverlb *k3d.Node
	for _, node := range nodes {
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
		match := NodeFilterRegexp.FindStringSubmatch(filter)

		if len(match) == 0 {
			return nil, fmt.Errorf("Failed to parse node filters: invalid format or empty subset in '%s'", filter)
		}

		// map capturing group names to submatches
		submatches := MapSubexpNames(NodeFilterRegexp.SubexpNames(), match)

		// error out if filter is specified (should only work in FilterNodesWithSuffix)
		if sf, ok := submatches["suffix"]; ok && sf != "" {
			return nil, fmt.Errorf("error filtering with '%s': no suffix allowed in simple filter", filter)
		}

		// if one of the filters is 'all', we only return this and drop all others
		if submatches["group"] == "all" {
			if len(filters) > 1 {
				l.Log().Warnf("Node filter 'all' set, but more were specified in '%+v'", filters)
			}
			return nodes, nil
		}

		// Choose the group of nodes to operate on
		groupNodes := []*k3d.Node{}
		if role, ok := rolesByIdentifier[submatches["group"]]; ok {
			switch role {
			case k3d.ServerRole:
				groupNodes = serverNodes
			case k3d.AgentRole:
				groupNodes = agentNodes
			case k3d.LoadBalancerRole:
				if serverlb == nil {
					return nil, fmt.Errorf("Node filter '%s' targets a node that does not exist (disabled?)", filter)
				}
				filteredNodes = append(filteredNodes, serverlb)
				return filteredNodes, nil // early exit if filtered group is the loadbalancer
			}
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
			 * subset specified by a range 'START-END', where each side is optional
			 */

			split := strings.Split(submatches["subsetRange"], "-")
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

	l.Log().Tracef("Filtered %d nodes (filter: %s)", len(filteredNodes), filters)

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
