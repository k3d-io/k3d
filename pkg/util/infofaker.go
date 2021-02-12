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
	"os"
	"path"
	"strings"

	dockerunits "github.com/docker/go-units"
)

// creates a mininal fake meminfo with fields required by cadvisor (see machine.go in cadvisor)
func meminfoContent(totalKB int64) string {
	var lines = []string{
		fmt.Sprintf("MemTotal: %d kB", totalKB),
		// this may be configurable later
		"SwapTotal: 0 kB",
	}
	return strings.Join(lines, "\n")
}

// FakeMeminfo creates a fake meminfo file to be mounted and provide a specific RAM capacity.
// This file is created on a per specific container basis, container id ensures that.
// Returns a path to the file
func FakeMeminfo(memoryBytes int64, containerID string) (string, error) {
	// this file needs to be kept across reboots, keep it in ~/.k3d
	configdir, err := GetConfigDirOrCreate()
	if err != nil {
		return "", err
	}
	fakeMeminfoFilename := fmt.Sprintf(".meminfo-%s", containerID)
	fakememinfo, err := os.Create(path.Join(configdir, fakeMeminfoFilename))
	defer fakememinfo.Close()
	if err != nil {
		return "", err
	}

	// write content, must be kB
	memoryKb := memoryBytes / dockerunits.KB
	content := meminfoContent(memoryKb)
	_, err = fakememinfo.WriteString(content)
	if err != nil {
		return "", err
	}

	return fakememinfo.Name(), nil
}
