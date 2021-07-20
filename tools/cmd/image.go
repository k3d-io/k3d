package run

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/client"
)

func imageSave(images []string, dest, clusterName string) error {
	// get a docker client
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("ERROR: couldn't create docker client\n%w", err)
	}

	imageReader, err := docker.ImageSave(ctx, images)
	if err != nil {
		return fmt.Errorf("ERROR: couldn't save images %s\n%w", images, err)
	}
	defer imageReader.Close()

	tarFileName := dest
	if !strings.HasSuffix(dest, ".tar") {
		if !strings.HasSuffix(dest, "/") {
			dest = dest + "/"
		}
		tarFileName = fmt.Sprintf("%sk3d-%s-images-%s.tar", dest, clusterName, time.Now().Format("20060102150405"))
	}
	tarFile, err := os.Create(tarFileName)
	if err != nil {
		return fmt.Errorf("ERROR: couldn't create tarfile [%s]\n%w", tarFileName, err)
	}
	defer tarFile.Close()

	if _, err = io.Copy(tarFile, imageReader); err != nil {
		return fmt.Errorf("ERROR: couldn't save image stream to tarfile\n%w", err)
	}

	log.Printf("INFO: saved images %s to [%s]", images, tarFileName)

	return nil
}
