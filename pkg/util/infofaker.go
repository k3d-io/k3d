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
	log "github.com/sirupsen/logrus"
)

const (
	EdacFolderPath = "/sys/devices/system/edac"
	MemInfoPath    = "/proc/meminfo"
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

// GetNodeFakerDirOrCreate creates or gets a hidden folder in k3d home dir
// to keep container(node)-specific fake files in it
func GetNodeFakerDirOrCreate(name string) (string, error) {
	// this folder needs to be kept across reboots, keep it in ~/.k3d
	configdir, err := GetConfigDirOrCreate()
	if err != nil {
		return "", err
	}
	fakeDir := path.Join(configdir, fmt.Sprintf(".%s", name))

	// create directories if necessary
	if err := createDirIfNotExists(fakeDir); err != nil {
		log.Errorf("Failed to create fake files path '%s'", fakeDir)
		return "", err
	}

	return fakeDir, nil

}

// GetFakeMeminfoPathForName returns a path to (existent or not) fake meminfo file for a given node/container name
func GetFakeMeminfoPathForName(nodeName string) (string, error) {
	return fakeInfoPathForName("meminfo", nodeName)
}

// MakeFakeMeminfo creates a fake meminfo file to be mounted and provide a specific RAM capacity.
// This file is created on a per specific container/node basis, uniqueName must ensure that.
// Returns a path to the file
func MakeFakeMeminfo(memoryBytes int64, nodeName string) (string, error) {
	fakeMeminfoPath, err := GetFakeMeminfoPathForName(nodeName)
	if err != nil {
		return "", err
	}
	fakememinfo, err := os.Create(fakeMeminfoPath)
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

// MakeFakeEdac creates an empty edac folder to force cadvisor
// to use meminfo even for ECC memory
func MakeFakeEdac(nodeName string) (string, error) {
	dir, err := GetNodeFakerDirOrCreate(nodeName)
	if err != nil {
		return "", err
	}
	edacPath := path.Join(dir, "edac")
	// create directories if necessary
	if err := createDirIfNotExists(edacPath); err != nil {
		log.Errorf("Failed to create fake edac path '%s'", edacPath)
		return "", err
	}

	return edacPath, nil
}

// returns a path to (existent or not) fake (mem or cpu)info file for a given node/container name
func fakeInfoPathForName(infoType string, nodeName string) (string, error) {
	// this file needs to be kept across reboots, keep it in ~/.k3d
	dir, err := GetNodeFakerDirOrCreate(nodeName)
	if err != nil {
		return "", err
	}
	return path.Join(dir, infoType), nil
}
