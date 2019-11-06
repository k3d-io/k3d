/*
Copyright © 2019 Thorsten Klein <iwilltry42@gmail.com>

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
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	k3d "github.com/rancher/k3d/pkg/types"

	"regexp"
)

// possible matches
// -
var filterRegexp = regexp.MustCompile(`^(?P<group>master|worker|all)(?P<subsetSpec>\[(?P<subset>(?P<subsetList>(\d+,?)+)|(?P<subsetRange>\d*:\d*)|(?P<subsetWildcard>\*))\])?$`)

// mapSubexpNames maps regex capturing group names to corresponding matches
func mapSubexpNames(names, matches []string) map[string]string {
	//names, matches = names[1:], matches[1:]
	nameMatchMap := make(map[string]string, len(matches))
	for index := range names {
		nameMatchMap[names[index]] = matches[index]
	}
	return nameMatchMap
}

// SplitFilterFromFlag separates a flag's value from the node filter, if there is one
func SplitFilterFromFlag(flag string) (string, string, error) {

	/* Case 1) no filter specified */

	if !strings.Contains(flag, "@") {
		return flag, "", nil
	}

	/* Case 2) filter indicated using '@' in flag */

	split := strings.Split(flag, "@")

	// max number of pieces after split = 2 (only one @ allowed in flag)
	if len(split) > 2 {
		return "", "", fmt.Errorf("Invalid flag '%s': only one '@' for node filter allowed", flag)
	}

	// trailing or leading '@'
	if len(split) < 2 {
		return "", "", fmt.Errorf("Invalid flag '%s' includes '@' but is missing either an object or a filter", flag)
	}

	return split[0], split[1], nil

}

// FilterNodes takes a string filter to return a filtered list of nodes
func FilterNodes(nodes *[]k3d.Node, filterString string) ([]*k3d.Node, error) {

	// filterString is a semicolon-separated list of node filters
	filters := strings.Split(filterString, ";")

	if len(filters) == 0 {
		return nil, fmt.Errorf("No filter specified")
	}

	// range over all instances of group[subset] specs
	for _, filter := range filters {

		/* Step 1: match regular expression */

		match := filterRegexp.FindStringSubmatch(filter)

		if len(match) == 0 {
			return nil, fmt.Errorf("Failed to parse node filters: invalid format or empty subset in '%s'", filter)
		}

		// map capturing group names to submatches
		submatches := mapSubexpNames(filterRegexp.SubexpNames(), match)

		log.Debugf("Matches: %+v", submatches)

		/* Step 2 - Evaluate */

		// if one of the filters is 'all', we only return this and drop all others
		if submatches["group"] == "all" {
			// TODO: only log if really more than one is specified
			log.Warnf("Node Specifier 'all' set, but more were specified in '%s'", filterString)
			return nodes, nil
		}

		/*  */
		subset := []int{}

		if submatches["subsetList"] != "" {
			for _, index := range strings.Split(submatches["subsetList"], ",") {
				if index != "" {
					num, err := strconv.Atoi(index)
					if err != nil {
						return rolesMap, fmt.Errorf("Failed to convert subset number to integer in '%s'", filter)
					}
					subset = append(subset, num)
				}
			}

		} else if submatches["subsetRange"] != "" {

			/*
			 * subset specified by a range 'START:END', where each side is optional
			 */

			split := strings.Split(submatches["subsetRange"], ":")
			if len(split) != 2 {
				return rolesMap, fmt.Errorf("Failed to parse subset range in '%s'", filter)
			}

			start := 0
			end := numNodes - 1
			var err error

			if split[0] != "" {
				start, err = strconv.Atoi(split[0])
				if err != nil {
					return rolesMap, fmt.Errorf("Failed to convert subset range start to integer in '%s'", filter)
				}
				if start < 0 {
					return rolesMap, fmt.Errorf("Invalid subset range: start < 0 in '%s'", filter)
				}
			}

			if split[1] != "" {
				end, err = strconv.Atoi(split[1])
				if err != nil {
					return rolesMap, fmt.Errorf("Failed to convert subset range start to integer in '%s'", filter)
				}
				if end < start {
					return rolesMap, fmt.Errorf("Invalid subset range: end < start in '%s'", filter)
				}
			}

			for i := start; i <= end; i++ {
				subset = append(subset, i)
			}

		} else if submatches["subsetWildcard"] == "*" {
			/*
			 * '*' = all nodes
			 */
			for i := 0; i < numNodes; i++ {
				subset = append(subset, i)
			}
		} else {
			return rolesMap, fmt.Errorf("Failed to parse node specifiers: unknown subset in '%s'", filter)
		}

		rolesMap[submatches["group"]] = append(rolesMap[submatches["group"]], subset...)

	}

	log.Debugf("ROLESMAP: %+v", rolesMap)

	return rolesMap, nil
}
