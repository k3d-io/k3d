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
package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	runtimeErrors "github.com/k3d-io/k3d/v5/pkg/runtimes/errors"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

// CreateVolume creates a new named volume
func (d Docker) CreateVolume(ctx context.Context, name string, labels map[string]string) error {
	// (0) create new docker client
	docker, err := GetDockerClient()
	if err != nil {
		return fmt.Errorf("failed to get docker client: %w", err)
	}
	defer docker.Close()

	// (1) create volume
	volumeCreateOptions := volume.CreateOptions{
		Name:       name,
		Labels:     labels,
		Driver:     "local", // TODO: allow setting driver + opts
		DriverOpts: map[string]string{},
	}

	for k, v := range k3d.DefaultRuntimeLabels {
		volumeCreateOptions.Labels[k] = v
	}
	for k, v := range k3d.DefaultRuntimeLabelsVar {
		volumeCreateOptions.Labels[k] = v
	}

	_, err = docker.VolumeCreate(ctx, volumeCreateOptions)
	if err != nil {
		return fmt.Errorf("failed to create volume '%s': %w", name, err)
	}
	return nil
}

// DeleteVolume creates a new named volume
func (d Docker) DeleteVolume(ctx context.Context, name string) error {
	// (0) create new docker client
	docker, err := GetDockerClient()
	if err != nil {
		return fmt.Errorf("failed to get docker client: %w", err)
	}
	defer docker.Close()

	// get volume and delete it
	vol, err := docker.VolumeInspect(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to find volume '%s': %w", name, err)
	}

	// check if volume is still in use
	if vol.UsageData != nil {
		if vol.UsageData.RefCount > 0 {
			return fmt.Errorf("failed to delete volume '%s' as it is still referenced by %d containers", name, vol.UsageData.RefCount)
		}
	}

	// remove volume
	if err := docker.VolumeRemove(ctx, name, true); err != nil {
		return fmt.Errorf("docker failed to delete volume '%s': %w", name, err)
	}

	return nil
}

// GetVolume tries to get a named volume
func (d Docker) GetVolume(name string) (string, error) {
	// (0) create new docker client
	ctx := context.Background()
	docker, err := GetDockerClient()
	if err != nil {
		return "", fmt.Errorf("failed to get docker client: %w", err)
	}
	defer docker.Close()

	filters := filters.NewArgs()
	filters.Add("name", fmt.Sprintf("^%s$", name))
	volumeList, err := docker.VolumeList(ctx, volume.ListOptions{Filters: filters})
	if err != nil {
		return "", fmt.Errorf("docker failed to list volumes: %w", err)
	}
	if len(volumeList.Volumes) < 1 {
		return "", fmt.Errorf("failed to find named volume '%s': %w", name, runtimeErrors.ErrRuntimeVolumeNotExists)
	}

	return volumeList.Volumes[0].Name, nil
}

func (d Docker) GetVolumesByLabel(ctx context.Context, labels map[string]string) ([]string, error) {
	var volumes []string
	// (0) create new docker client
	docker, err := GetDockerClient()
	if err != nil {
		return volumes, fmt.Errorf("failed to get docker client: %w", err)
	}
	defer docker.Close()

	// (1) list containers which have the default k3d labels attached
	filters := filters.NewArgs()
	for k, v := range k3d.DefaultRuntimeLabels {
		filters.Add("label", fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range labels {
		filters.Add("label", fmt.Sprintf("%s=%s", k, v))
	}

	volumeList, err := docker.VolumeList(ctx, volume.ListOptions{Filters: filters})
	if err != nil {
		return volumes, fmt.Errorf("docker failed to list volumes: %w", err)
	}

	for _, v := range volumeList.Volumes {
		volumes = append(volumes, v.Name)
	}

	return volumes, nil
}
