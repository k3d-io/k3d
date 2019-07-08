package run

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

const imageBasePathRemote = "/images"

func importImage(clusterName string, images []string) error {
	// get a docker client
	ctx := context.Background()
	docker, err := client.NewEnvClient()
	if err != nil {
		return fmt.Errorf("ERROR: couldn't create docker client\n%+v", err)
	}

	// get cluster directory to temporarily save the image tarball there
	imageVolume, err := getImageVolume(clusterName)
	if err != nil {
		return fmt.Errorf("ERROR: couldn't get image volume for cluster [%s]\n%+v", clusterName, err)
	}

	//*** first, save the images using the local docker daemon
	log.Printf("INFO: Saving images [%s] from local docker daemon...", images)
	imageReader, err := docker.ImageSave(ctx, images)
	if err != nil {
		return fmt.Errorf("ERROR: failed to save images [%s] locally\n%+v", images, err)
	}

	// TODO: create tar from stream
	tmpFile, err := ioutil.TempFile("", "*.tar")
	if err != nil {
		return fmt.Errorf("ERROR: couldn't create temp file to cache tarball\n%+v", err)
	}
	defer tmpFile.Close()
	if _, err = io.Copy(tmpFile, imageReader); err != nil {
		return fmt.Errorf("ERROR: couldn't write image stream to tar [%s]\n%+v", tmpFile.Name(), err)
	}

	// create a dummy container to get the tarball into the named volume
	containerConfig := container.Config{
		Hostname: "k3d-dummy", // TODO: change details here
		Image:    "rancher/k3s:v0.7.0-rc2",
		Labels: map[string]string{
			"app":     "k3d",
			"cluster": "test",
		},
	}
	hostConfig := container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:%s:rw", imageVolume.Name, imageBasePathRemote),
		},
	}
	dummyContainer, err := docker.ContainerCreate(ctx, &containerConfig, &hostConfig, &network.NetworkingConfig{}, "k3d-dummy")
	if err != nil {
		return fmt.Errorf("ERROR: couldn't create dummy container\n%+v", err)
	}

	fmt.Println(ioutil.ReadAll(imageReader))
	if err = docker.CopyToContainer(ctx, dummyContainer.ID, "/images", imageReader, types.CopyToContainerOptions{}); err != nil {
		return fmt.Errorf("ERROR: couldn't copy tarball to dummy container\n%+v", err)
	}

	if err = docker.ContainerRemove(ctx, "k3d-dummy", types.ContainerRemoveOptions{
		Force: true,
	}); err != nil {
		return fmt.Errorf("ERROR: couldn't remove dummy container\n%+v", err)
	}

	// Get the container IDs for all containers in the cluster
	clusters, err := getClusters(false, clusterName)
	if err != nil {
		return fmt.Errorf("ERROR: couldn't get cluster by name [%s]\n%+v", clusterName, err)
	}
	containerList := []types.Container{clusters[clusterName].server}
	containerList = append(containerList, clusters[clusterName].workers...)

	// *** second, import the images using ctr in the k3d nodes

	// create exec configuration
	command := fmt.Sprintf("ctr image import %s", imageBasePathRemote+"/test.tar")
	cmd := []string{"sh", "-c", command}
	execConfig := types.ExecConfig{
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          cmd,
		Tty:          true,
		Detach:       true,
	}

	execAttachConfig := types.ExecConfig{
		Tty: true,
	}

	execStartConfig := types.ExecStartCheck{
		Tty: true,
	}

	// import in each node separately
	// TODO: create a shared image cache volume, so we don't need to import it separately
	for _, container := range containerList {

		containerName := container.Names[0][1:] // trimming the leading "/" from name
		log.Printf("INFO: Importing image [%s] in container [%s]", images, containerName)

		// create exec configuration
		execResponse, err := docker.ContainerExecCreate(ctx, container.ID, execConfig)
		if err != nil {
			return fmt.Errorf("ERROR: Failed to create exec command for container [%s]\n%+v", containerName, err)
		}

		// attach to exec process in container
		containerConnection, err := docker.ContainerExecAttach(ctx, execResponse.ID, execAttachConfig)
		if err != nil {
			return fmt.Errorf("ERROR: couldn't attach to container [%s]\n%+v", containerName, err)
		}
		defer containerConnection.Close()

		// start exec
		err = docker.ContainerExecStart(ctx, execResponse.ID, execStartConfig)
		if err != nil {
			return fmt.Errorf("ERROR: couldn't execute command in container [%s]\n%+v", containerName, err)
		}

		// get output from container
		content, err := ioutil.ReadAll(containerConnection.Reader)
		if err != nil {
			return fmt.Errorf("ERROR: couldn't read output from container [%s]\n%+v", containerName, err)
		}

		// example output "unpacking image........ ...done"
		if !strings.Contains(string(content), "done") {
			return fmt.Errorf("ERROR: seems like something went wrong using `ctr image import` in container [%s]. Full output below:\n%s", containerName, string(content))
		}
	}

	log.Printf("INFO: Successfully imported image [%s] in all nodes of cluster [%s]", images, clusterName)

	log.Println("INFO: Cleaning up tarball...")
	/*if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("ERROR: Couldn't close tarfile [%s]\n%+v", tmpFile.Name(), err)
	}
	if err = os.Remove(tmpFile.Name()); err != nil {
		return fmt.Errorf("ERROR: Couldn't remove tarball [%s]\n%+v", tmpFile.Name(), err)
	}*/
	log.Println("INFO: ...Done")

	return nil
}
