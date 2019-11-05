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
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

// ValidateVolumeFlag checks, if the source of volume mounts exists and if the destination is an absolute path
// a flag can look like this: --volume SRC:DST@NODES, where...
// - SRC: source directory/file -> tests: must exist
// - DEST: source directory/file -> tests: must be absolute path
// - NODES: specifies a set of nodes -> tests: fit regex // TODO:
func ValidateVolumeFlag(volumeFlags []string) (string, string, error) {

	for _, volume := range volumeFlags {
		src := ""
		dest := ""
		nodes := ""

		split := strings.Split(volume, "@")

		// max number of pieces after split = 2 (only one @ allowed in flag)
		if len(split) > 2 {
			return rolesToVolumesMap, fmt.Errorf("Invalid volume flag '%s': only one '@' allowed", volume)
		}

		// min number of pieces after split = 1 (catch empty flags)
		if len(split) > 1 {
			nodes = split[1]
		}

		// catch all other unlikely cases
		if len(split) == 0 || split[0] == "" {
			return rolesToVolumesMap, fmt.Errorf("Invalid volume flag '%s'", volume)
		}

		// validate 'SRC[:DEST]' substring
		split = strings.Split(split[0], ":")
		if len(split) < 1 || len(split) > 2 {
			return rolesToVolumesMap, fmt.Errorf("Invalid volume mount '%s': only one ':' allowed", volume)
		}

		// we only have SRC specified -> DEST = SRC
		if len(split) == 1 {
			src = split[0]
			dest = src
		} else {
			src = split[0]
			dest = split[1]
		}

		// verify that the source exists
		if src != "" {
			if _, err := os.Stat(src); err != nil {
				return rolesToVolumesMap, fmt.Errorf("Failed to stat file/dir that you're trying to mount: '%s' in '%s'", src, volume)
			}
		}

		// verify that the destination is an absolute path
		if !strings.HasPrefix(dest, "/") {
			return rolesToVolumesMap, fmt.Errorf("Volume mount destination doesn't appear to be an absolute path: '%s' in '%s'", dest, volume)
		}

		// put into struct
		volumeSpec.Source = src
		volumeSpec.Destination = dest

		// if no nodes are specified with an @nodeset, attach volume to all nodes and go to next value
		if nodes == "" {
			rolesToVolumesMap.AllRoles = append(rolesToVolumesMap.AllRoles, volumeSpec)
			continue
		}

		specifiedNodes, err := parseNodeSpecifier(nodes, masterCount, workerCount)
		if err != nil {
			log.Errorf("Failed to parse node specifier on '--volume %s'", volume)
			return rolesToVolumesMap, err
		}
		log.Debugf("Specified nodes: %+v", specifiedNodes)

	}
	return rolesToVolumesMap, nil
}
