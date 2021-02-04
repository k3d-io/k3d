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
package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	log "github.com/sirupsen/logrus"
)

// CreateVolume creates a new named volume
func (d Docker) CreateVolume(ctx context.Context, name string, labels map[string]string) error {
	// (0) create new docker client
	docker, err := GetDockerClient()
	if err != nil {
		log.Errorln("Failed to create docker client")
		return err
	}
	defer docker.Close()

	// (1) create volume
	volumeCreateOptions := volume.VolumeCreateBody{
		Name:       name,
		Labels:     labels,
		Driver:     "local", // TODO: allow setting driver + opts
		DriverOpts: map[string]string{},
	}

	for k, v := range k3d.DefaultObjectLabels {
		volumeCreateOptions.Labels[k] = v
	}

	vol, err := docker.VolumeCreate(ctx, volumeCreateOptions)
	if err != nil {
		log.Errorf("Failed to create volume '%s'", name)
		return err
	}
	log.Infof("Created volume '%s'", vol.Name)
	return nil
}

// DeleteVolume creates a new named volume
func (d Docker) DeleteVolume(ctx context.Context, name string) error {
	// (0) create new docker client
	docker, err := GetDockerClient()
	if err != nil {
		log.Errorln("Failed to create docker client")
		return err
	}
	defer docker.Close()

	// get volume and delete it
	vol, err := docker.VolumeInspect(ctx, name)
	if err != nil {
		log.Errorf("Failed to find volume '%s'", name)
		return err
	}

	// check if volume is still in use
	if vol.UsageData != nil {
		if vol.UsageData.RefCount > 0 {
			log.Errorf("Failed to delete volume '%s'", vol.Name)
			return fmt.Errorf("Volume '%s' is still referenced by %d containers", name, vol.UsageData.RefCount)
		}
	}

	// remove volume
	if err := docker.VolumeRemove(ctx, name, true); err != nil {
		log.Errorf("Failed to delete volume '%s'", name)
		return err
	}

	return nil
}

// GetVolume tries to get a named volume
func (d Docker) GetVolume(name string) (string, error) {
	// (0) create new docker client
	ctx := context.Background()
	docker, err := GetDockerClient()
	if err != nil {
		log.Errorln("Failed to create docker client")
		return "", err
	}
	defer docker.Close()

	filters := filters.NewArgs()
	filters.Add("name", fmt.Sprintf("^%s$", name))
	volumeList, err := docker.VolumeList(ctx, filters)
	if err != nil {
		return "", err
	}
	if len(volumeList.Volumes) < 1 {
		return "", fmt.Errorf("Failed to find named volume '%s'", name)
	}

	return volumeList.Volumes[0].Name, nil

}
