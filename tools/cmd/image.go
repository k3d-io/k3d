package run

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/docker/docker/client"
)

func imageSave(images []string, dest, clusterName string) error {
	// get a docker client
	ctx := context.Background()
	docker, err := client.NewEnvClient()
	if err != nil {
		return fmt.Errorf("ERROR: couldn't create docker client\n%+v", err)
	}

	imageReader, err := docker.ImageSave(ctx, images)
	if err != nil {
		return fmt.Errorf("ERROR: couldn't save images %s\n%+v", images, err)
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
		return fmt.Errorf("ERROR: couldn't create tarfile [%s]\n%+v", tarFileName, err)
	}
	defer tarFile.Close()

	if _, err = io.Copy(tarFile, imageReader); err != nil {
		return fmt.Errorf("ERROR: couldn't save image stream to tarfile\n%+v", err)
	}

	log.Printf("INFO: saved images %s to [%s]", images, tarFileName)

	return nil
}
