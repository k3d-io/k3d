package run

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

type Volumes struct {
	DefaultVolumes       []string
	NodeSpecificVolumes  map[string][]string
	GroupSpecificVolumes map[string][]string
}

// createImageVolume will create a new docker volume used for storing image tarballs that can be loaded into the clusters
func createImageVolume(clusterName string) (types.Volume, error) {

	var vol types.Volume

	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return vol, fmt.Errorf(" Couldn't create docker client\n%+v", err)
	}

	volName := fmt.Sprintf("k3d-%s-images", clusterName)

	volumeCreateOptions := volume.VolumeCreateBody{
		Name: volName,
		Labels: map[string]string{
			"app":     "k3d",
			"cluster": clusterName,
		},
		Driver:     "local", //TODO: allow setting driver + opts
		DriverOpts: map[string]string{},
	}
	vol, err = docker.VolumeCreate(ctx, volumeCreateOptions)
	if err != nil {
		return vol, fmt.Errorf("failed to create image volume [%s] for cluster [%s]\n%+v", volName, clusterName, err)
	}

	return vol, nil
}

// deleteImageVolume will delete the volume we created for sharing images with this cluster
func deleteImageVolume(clusterName string) error {

	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf(" Couldn't create docker client\n%+v", err)
	}

	volName := fmt.Sprintf("k3d-%s-images", clusterName)

	if err = docker.VolumeRemove(ctx, volName, true); err != nil {
		return fmt.Errorf(" Couldn't remove volume [%s] for cluster [%s]\n%+v", volName, clusterName, err)
	}

	return nil
}

// getImageVolume returns the docker volume object representing the imagevolume for the cluster
func getImageVolume(clusterName string) (types.Volume, error) {
	var vol types.Volume
	volName := fmt.Sprintf("k3d-%s-images", clusterName)

	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return vol, fmt.Errorf(" Couldn't create docker client\n%+v", err)
	}

	filters := filters.NewArgs()
	filters.Add("label", "app=k3d")
	filters.Add("label", fmt.Sprintf("cluster=%s", clusterName))
	volumeList, err := docker.VolumeList(ctx, filters)
	if err != nil {
		return vol, fmt.Errorf(" Couldn't get volumes for cluster [%s]\n%+v ", clusterName, err)
	}
	volFound := false
	for _, volume := range volumeList.Volumes {
		if volume.Name == volName {
			vol = *volume
			volFound = true
			break
		}
	}
	if !volFound {
		return vol, fmt.Errorf("didn't find volume [%s] in list of volumes returned for cluster [%s]", volName, clusterName)
	}

	return vol, nil
}

func NewVolumes(volumes []string) (*Volumes, error) {
	volumesSpec := &Volumes{
		DefaultVolumes:       []string{},
		NodeSpecificVolumes:  make(map[string][]string),
		GroupSpecificVolumes: make(map[string][]string),
	}

volumes:
	for _, volume := range volumes {
		if strings.Contains(volume, "@") {
			split := strings.Split(volume, "@")
			if len(split) != 2 {
				return nil, fmt.Errorf("invalid node volume spec: %s", volume)
			}

			nodeVolumes := split[0]
			node := strings.ToLower(split[1])
			if len(node) == 0 {
				return nil, fmt.Errorf("invalid node volume spec: %s", volume)
			}

			// check if node selector is a node group
			for group, names := range nodeRuleGroupsMap {
				added := false

				for _, name := range names {
					if name == node {
						volumesSpec.addGroupSpecificVolume(group, nodeVolumes)
						added = true
						break
					}
				}

				if added {
					continue volumes
				}
			}

			// otherwise this is a volume for a specific node
			volumesSpec.addNodeSpecificVolume(node, nodeVolumes)
		} else {
			volumesSpec.DefaultVolumes = append(volumesSpec.DefaultVolumes, volume)
		}
	}

	return volumesSpec, nil
}

// addVolumesToHostConfig adds all default volumes and node / group specific volumes to a HostConfig
func (v Volumes) addVolumesToHostConfig(containerName string, groupName string, hostConfig *container.HostConfig) {
	volumes := v.DefaultVolumes

	if v, ok := v.NodeSpecificVolumes[containerName]; ok {
		volumes = append(volumes, v...)
	}

	if v, ok := v.GroupSpecificVolumes[groupName]; ok {
		volumes = append(volumes, v...)
	}

	if len(volumes) > 0 {
		hostConfig.Binds = volumes
	}
}

func (v *Volumes) addNodeSpecificVolume(node, volume string) {
	if _, ok := v.NodeSpecificVolumes[node]; !ok {
		v.NodeSpecificVolumes[node] = []string{}
	}
	v.NodeSpecificVolumes[node] = append(v.NodeSpecificVolumes[node], volume)
}

func (v *Volumes) addGroupSpecificVolume(group, volume string) {
	if _, ok := v.GroupSpecificVolumes[group]; !ok {
		v.GroupSpecificVolumes[group] = []string{}
	}
	v.GroupSpecificVolumes[group] = append(v.GroupSpecificVolumes[group], volume)
}
