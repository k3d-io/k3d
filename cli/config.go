package run

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path"

	dockerClient "github.com/docker/docker/client"
	"github.com/mitchellh/go-homedir"
	"github.com/olekukonko/tablewriter"
)

type cluster struct {
	name   string
	image  string
	status string
}

// createDirIfNotExists checks for the existence of a directory and creates it along with all required parents if not.
// It returns an error if the directory (or parents) couldn't be created and nil if it worked fine or if the path already exists.
func createDirIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, os.ModePerm)
	}
	return nil
}

// createClusterDir creates a directory with the cluster name under $HOME/.config/k3d/<cluster_name>.
// The cluster directory will be used e.g. to store the kubeconfig file.
func createClusterDir(name string) {
	clusterPath, _ := getClusterDir(name)
	if err := createDirIfNotExists(clusterPath); err != nil {
		log.Fatalf("ERROR: couldn't create cluster directory [%s] -> %+v", clusterPath, err)
	}
}

// deleteClusterDir contrary to createClusterDir, this deletes the cluster directory under $HOME/.config/k3d/<cluster_name>
func deleteClusterDir(name string) {
	clusterPath, _ := getClusterDir(name)
	if err := os.RemoveAll(clusterPath); err != nil {
		log.Printf("WARNING: couldn't delete cluster directory [%s]. You might want to delete it manually.", clusterPath)
	}
}

// getClusterDir returns the path to the cluster directory which is $HOME/.config/k3d/<cluster_name>
func getClusterDir(name string) (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		log.Printf("ERROR: Couldn't get user's home directory")
		return "", err
	}
	return path.Join(homeDir, ".config", "k3d", name), nil
}

// printClusters prints the names of existing clusters
func printClusters(all bool) {
	clusterNames, err := getClusterNames()
	if err != nil {
		log.Fatalf("ERROR: Couldn't list clusters -> %+v", err)
	}
	if len(clusterNames) == 0 {
		log.Printf("No clusters found!")
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"NAME", "IMAGE", "STATUS"})

	for _, clusterName := range clusterNames {
		cluster, _ := getCluster(clusterName)
		clusterData := []string{cluster.name, cluster.image, cluster.status}
		if cluster.status == "running" || all {
			table.Append(clusterData)
		}
	}
	table.Render()
}

// getClusterNames returns a list of cluster names which are folder names in the config directory
func getClusterNames() ([]string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		log.Printf("ERROR: Couldn't get user's home directory")
		return nil, err
	}
	configDir := path.Join(homeDir, ".config", "k3d")
	files, err := ioutil.ReadDir(configDir)
	if err != nil {
		log.Printf("ERROR: Couldn't list files in [%s]", configDir)
		return nil, err
	}
	clusters := []string{}
	for _, file := range files {
		if file.IsDir() {
			clusters = append(clusters, file.Name())
		}
	}
	return clusters, nil
}

// getCluster creates a cluster struct with populated information fields
func getCluster(name string) (cluster, error) {
	cluster := cluster{
		name:   name,
		image:  "UNKNOWN",
		status: "UNKNOWN",
	}

	docker, err := dockerClient.NewEnvClient()
	if err != nil {
		log.Printf("ERROR: couldn't create docker client -> %+v", err)
		return cluster, err
	}
	containerInfo, err := docker.ContainerInspect(context.Background(), cluster.name)
	if err != nil {
		log.Printf("WARNING: couldn't get docker info for [%s] -> %+v", cluster.name, err)
	} else {
		cluster.image = containerInfo.Config.Image
		cluster.status = containerInfo.ContainerJSONBase.State.Status
	}
	return cluster, nil
}
