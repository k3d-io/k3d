package run

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

const (
	imageBasePathRemote = "/images"
	k3dToolsImage       = "docker.io/iwilltry42/k3d-tools:v0.0.1"
)

func importImage(clusterName string, images []string, noRemove bool) error {
	// get a docker client
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf(" Couldn't create docker client\n%+v", err)
	}

	// get cluster directory to temporarily save the image tarball there
	imageVolume, err := getImageVolume(clusterName)
	if err != nil {
		return fmt.Errorf(" Couldn't get image volume for cluster [%s]\n%+v", clusterName, err)
	}

	//*** first, save the images using the local docker daemon
	log.Infof("Saving images %s from local docker daemon...", images)
	toolsContainerName := fmt.Sprintf("k3d-%s-tools", clusterName)
	tarFileName := fmt.Sprintf("%s/k3d-%s-images-%s.tar", imageBasePathRemote, clusterName, time.Now().Format("20060102150405"))

	// create a tools container to get the tarball into the named volume
	containerConfig := container.Config{
		Hostname: toolsContainerName,
		Image:    k3dToolsImage,
		Labels: map[string]string{
			"app":       "k3d",
			"cluster":   clusterName,
			"component": "tools",
		},
		Cmd:          append([]string{"save-image", "-d", tarFileName}, images...),
		AttachStdout: true,
		AttachStderr: true,
	}
	hostConfig := container.HostConfig{
		Binds: []string{
			"/var/run/docker.sock:/var/run/docker.sock",
			fmt.Sprintf("%s:%s:rw", imageVolume.Name, imageBasePathRemote),
		},
	}

	toolsContainerID, err := createContainer(&containerConfig, &hostConfig, &network.NetworkingConfig{}, toolsContainerName)
	if err != nil {
		return err
	}
	if err := startContainer(toolsContainerID); err != nil {
		return fmt.Errorf(" Couldn't start container %s\n%w", toolsContainerName, err)
	}

	defer func() {
		if err = docker.ContainerRemove(ctx, toolsContainerID, types.ContainerRemoveOptions{
			Force: true,
		}); err != nil {
			log.Warningf("Couldn't remove tools container\n%+v", err)
		}
	}()

	// loop to wait for tools container to exit (failed or successfully saved images)
	for {
		cont, err := docker.ContainerInspect(ctx, toolsContainerID)
		if err != nil {
			return fmt.Errorf(" Couldn't get helper container's exit code\n%+v", err)
		}
		if !cont.State.Running { // container finished...
			if cont.State.ExitCode == 0 { // ...successfully
				log.Info("Saved images to shared docker volume")
				break
			} else if cont.State.ExitCode != 0 { // ...failed
				errTxt := "Helper container failed to save images"
				logReader, err := docker.ContainerLogs(ctx, toolsContainerID, types.ContainerLogsOptions{
					ShowStdout: true,
					ShowStderr: true,
				})
				if err != nil {
					return fmt.Errorf("%s\n> couldn't get logs from helper container\n%+v", errTxt, err)
				}
				logs, err := ioutil.ReadAll(logReader) // let's show somw logs indicating what happened
				if err != nil {
					return fmt.Errorf("%s\n> couldn't get logs from helper container\n%+v", errTxt, err)
				}
				return fmt.Errorf("%s -> Logs from [%s]:\n>>>>>>\n%s\n<<<<<<", errTxt, toolsContainerName, string(logs))
			}
		}
		time.Sleep(time.Second / 2) // wait for half a second so we don't spam the docker API too much
	}

	// Get the container IDs for all containers in the cluster
	clusters, err := getClusters(false, clusterName)
	if err != nil {
		return fmt.Errorf(" Couldn't get cluster by name [%s]\n%+v", clusterName, err)
	}
	containerList := []types.Container{clusters[clusterName].server}
	containerList = append(containerList, clusters[clusterName].workers...)

	// *** second, import the images using ctr in the k3d nodes

	// create exec configuration
	cmd := []string{"ctr", "image", "import", tarFileName}
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
	// TODO: import concurrently using goroutines or find a way to share the image cache
	for _, container := range containerList {

		containerName := container.Names[0][1:] // trimming the leading "/" from name
		log.Infof("Importing images %s in container [%s]", images, containerName)

		// create exec configuration
		execResponse, err := docker.ContainerExecCreate(ctx, container.ID, execConfig)
		if err != nil {
			return fmt.Errorf("Failed to create exec command for container [%s]\n%+v", containerName, err)
		}

		// attach to exec process in container
		containerConnection, err := docker.ContainerExecAttach(ctx, execResponse.ID, types.ExecStartCheck{
			Detach: execAttachConfig.Detach,
			Tty:    execAttachConfig.Tty,
		})
		if err != nil {
			return fmt.Errorf(" Couldn't attach to container [%s]\n%+v", containerName, err)
		}
		defer containerConnection.Close()

		// start exec
		err = docker.ContainerExecStart(ctx, execResponse.ID, execStartConfig)
		if err != nil {
			return fmt.Errorf(" Couldn't execute command in container [%s]\n%+v", containerName, err)
		}

		// get output from container
		content, err := ioutil.ReadAll(containerConnection.Reader)
		if err != nil {
			return fmt.Errorf(" Couldn't read output from container [%s]\n%+v", containerName, err)
		}

		// example output "unpacking image........ ...done"
		if !strings.Contains(string(content), "done") {
			return fmt.Errorf("seems like something went wrong using `ctr image import` in container [%s]. Full output below:\n%s", containerName, string(content))
		}
	}

	log.Infof("Successfully imported images %s in all nodes of cluster [%s]", images, clusterName)

	// remove tarball from inside the server container
	if !noRemove {
		log.Info("Cleaning up tarball")

		execID, err := docker.ContainerExecCreate(ctx, clusters[clusterName].server.ID, types.ExecConfig{
			Cmd: []string{"rm", "-f", tarFileName},
		})
		if err != nil {
			log.Warningf("Failed to delete tarball: couldn't create remove in container [%s]\n%+v", clusters[clusterName].server.ID, err)
		}
		err = docker.ContainerExecStart(ctx, execID.ID, types.ExecStartCheck{
			Detach: true,
		})
		if err != nil {
			log.Warningf("Couldn't start tarball deletion action\n%+v", err)
		}

		for {
			execInspect, err := docker.ContainerExecInspect(ctx, execID.ID)
			if err != nil {
				log.Warningf("Couldn't verify deletion of tarball\n%+v", err)
			}

			if !execInspect.Running {
				if execInspect.ExitCode == 0 {
					log.Info("Deleted tarball")
					break
				} else {
					log.Warning("Failed to delete tarball")
					break
				}
			}
		}
	}

	log.Info("...Done")

	return nil
}
