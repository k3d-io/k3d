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
	"errors"
	"fmt"
	"os"
	rt "runtime"
	"strings"

	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	runtimeErrors "github.com/k3d-io/k3d/v5/pkg/runtimes/errors"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
)

// ValidateVolumeMount checks, if the source of volume mounts exists and if the destination is an absolute path
// - SRC: source directory/file -> tests: must exist
// - DEST: source directory/file -> tests: must be absolute path
func ValidateVolumeMount(ctx context.Context, runtime runtimes.Runtime, volumeMount string, cluster *k3d.Cluster) error {
	src, dest, err := ReadVolumeMount(volumeMount)
	if err != nil {
		return err
	}

	// verify that the source exists
	if src != "" {
		// directory/file: path containing / or \ (not allowed in named volumes)
		if strings.ContainsAny(src, "/\\") {
			if _, err := os.Stat(src); err != nil {
				l.Log().Warnf("failed to stat file/directory '%s' volume mount '%s': please make sure it exists", src, volumeMount)
			}
		} else {
			err := verifyNamedVolume(runtime, src)
			if err != nil {
				l.Log().Traceln(err)
				if errors.Is(err, runtimeErrors.ErrRuntimeVolumeNotExists) {
					if strings.HasPrefix(src, "k3d-") {
						if err := runtime.CreateVolume(ctx, src, map[string]string{k3d.LabelClusterName: cluster.Name}); err != nil {
							return fmt.Errorf("failed to create named volume '%s': %v", src, err)
						}
						cluster.Volumes = append(cluster.Volumes, src)
						l.Log().Infof("Created named volume '%s'", src)
					} else {
						l.Log().Infof("No named volume '%s' found. The runtime will create it automatically.", src)
					}
				} else {
					l.Log().Warnf("failed to get named volume: %v", err)
				}
			}
		}
	}

	// verify that the destination is an absolute path
	if !strings.HasPrefix(dest, "/") {
		return fmt.Errorf("volume mount destination doesn't appear to be an absolute path: '%s' in '%s'", dest, volumeMount)
	}

	return nil
}

func ReadVolumeMount(volumeMount string) (string, string, error) {
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
		return src, dest, fmt.Errorf("no volume/path specified")
	}
	if len(split) > maxParts {
		return src, dest, fmt.Errorf("invalid volume mount '%s': maximal %d ':' allowed", volumeMount, maxParts-1)
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
	return src, dest, nil
}

// verifyNamedVolume checks whether a named volume exists in the runtime
func verifyNamedVolume(runtime runtimes.Runtime, volumeName string) error {
	_, err := runtime.GetVolume(volumeName)
	if err != nil {
		return fmt.Errorf("runtime failed to get volume '%s': %w", volumeName, err)
	}
	return nil
}
