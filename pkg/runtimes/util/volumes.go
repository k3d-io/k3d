/*
Copyright © 2020-2021 The k3d Author(s)

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
	rt "runtime"
	"strings"

	"github.com/rancher/k3d/v5/pkg/runtimes"

	l "github.com/rancher/k3d/v5/pkg/logger"
)

// ValidateVolumeMount checks, if the source of volume mounts exists and if the destination is an absolute path
// - SRC: source directory/file -> tests: must exist
// - DEST: source directory/file -> tests: must be absolute path
func ValidateVolumeMount(runtime runtimes.Runtime, volumeMount string) error {
	src := ""
	dest := ""

	// validate 'SRC[:DEST]' substring
	split := strings.Split(volumeMount, ":")
	// a volume mapping can have 3 parts seperated by a ':' followed by a node filter
	// [SOURCE:]DEST[:OPT[,OPT]][@NODEFILTER[;NODEFILTER...]]
	// On Windows the source path needs to be an absolute path which means the path starts with
	// a drive designator and will also have a ':' in it. So for Windows the maxParts is increased by one.
	maxParts := 3
	if rt.GOOS == "windows" {
		maxParts++
	}
	if len(split) < 1 {
		return fmt.Errorf("No volume/path specified")
	}
	if len(split) > maxParts {
		return fmt.Errorf("Invalid volume mount '%s': maximal %d ':' allowed", volumeMount, maxParts-1)
	}

	// we only have SRC specified -> DEST = SRC
	// On windows the first part of the SRC is the drive letter, so we need to concat the first and second parts to get the path.
	if len(split) == 1 {
		src = split[0]
		dest = src
	} else if rt.GOOS == "windows" && len(split) >= 3 {
		src = split[0] + ":" + split[1]
		dest = split[2]
	} else {
		src = split[0]
		dest = split[1]
	}

	// verify that the source exists
	if src != "" {
		// a) named volume
		isNamedVolume := true
		if err := verifyNamedVolume(runtime, src); err != nil {
			isNamedVolume = false
		}
		if !isNamedVolume {
			if _, err := os.Stat(src); err != nil {
				l.Log().Warnf("Failed to stat file/directory/named volume that you're trying to mount: '%s' in '%s' -> Please make sure it exists", src, volumeMount)
			}
		}
	}

	// verify that the destination is an absolute path
	if !strings.HasPrefix(dest, "/") {
		return fmt.Errorf("Volume mount destination doesn't appear to be an absolute path: '%s' in '%s'", dest, volumeMount)
	}

	return nil
}

// verifyNamedVolume checks whether a named volume exists in the runtime
func verifyNamedVolume(runtime runtimes.Runtime, volumeName string) error {
	foundVolName, err := runtime.GetVolume(volumeName)
	if err != nil {
		return fmt.Errorf("runtime failed to get volume '%s': %w", volumeName, err)
	}
	if foundVolName == "" {
		return fmt.Errorf("failed to find named volume '%s'", volumeName)
	}
	return nil
}
