package run

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const imageBasePathRemote = "/images/"

func importImage(clusterName, image string) error {
	// get a docker client
	ctx := context.Background()
	docker, err := client.NewEnvClient()
	if err != nil {
		return fmt.Errorf("ERROR: couldn't create docker client\n%+v", err)
	}

	// get cluster directory to temporarily save the image tarball there
	imageBasePathLocal, err := getClusterDir(clusterName)
	imageBasePathLocal = imageBasePathLocal + "/images/"
	if err != nil {
		return fmt.Errorf("ERROR: couldn't get cluster directory for cluster [%s]\n%+v", clusterName, err)
	}

	// TODO: extend to enable importing a list of images
	imageList := []string{image}

	//*** first, save the images using the local docker daemon
	log.Printf("INFO: Saving image [%s] from local docker daemon...", image)
	imageReader, err := docker.ImageSave(ctx, imageList)
	if err != nil {
		return fmt.Errorf("ERROR: failed to save image [%s] locally\n%+v", image, err)
	}

	// create tarball
	imageTarName := strings.ReplaceAll(strings.ReplaceAll(image, ":", "_"), "/", "_") + ".tar"
	imageTar, err := os.Create(imageBasePathLocal + imageTarName)
	if err != nil {
		return err
	}
	defer imageTar.Close()

	_, err = io.Copy(imageTar, imageReader)
	if err != nil {
		return fmt.Errorf("ERROR: couldn't save image [%s] to file [%s]\n%+v", image, imageTar.Name(), err)
	}

	// TODO: get correct container ID by cluster name
	clusters, err := getClusters(false, clusterName)
	if err != nil {
		return fmt.Errorf("ERROR: couldn't get cluster by name [%s]\n%+v", clusterName, err)
	}
	containerList := []types.Container{clusters[clusterName].server}
	containerList = append(containerList, clusters[clusterName].workers...)

	// *** second, import the images using ctr in the k3d nodes

	// create exec configuration
	cmd := []string{"ctr", "image", "import", imageBasePathRemote + imageTarName}
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
		log.Printf("INFO: Importing image [%s] in container [%s]", image, containerName)

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

	log.Printf("INFO: Successfully imported image [%s] in all nodes of cluster [%s]", image, clusterName)

	log.Println("INFO: Cleaning up tarball...")
	if err := os.Remove(imageBasePathLocal + imageTarName); err != nil {
		return fmt.Errorf("ERROR: Couldn't remove tarball [%s]\n%+v", imageBasePathLocal+imageTarName, err)
	}
	log.Println("INFO: ...Done")

	return nil
}
