package run

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const defaultRegistryContainerName = "k3d-registry"

const defaultRegistryImage = "registry:2"

// Default registry port, both for the external and the internal ports
// Note well, that the internal port is never changed.
const defaultRegistryPort = 5000

const defaultFullRegistriesPath = "/etc/rancher/k3s/registries.yaml"

var defaultRegistryLabels = map[string]string{
	"app":       "k3d",
	"component": "registry",
}

// NOTE: structs copied from https://github.com/rancher/k3s/blob/master/pkg/agent/templates/registry.go
//       for avoiding a dependencies nightmare...

// Registry is registry settings configured
type Registry struct {
	// Mirrors are namespace to mirror mapping for all namespaces.
	Mirrors map[string]Mirror `toml:"mirrors" yaml:"mirrors"`

	// Configs are configs for each registry.
	// The key is the FDQN or IP of the registry.
	Configs map[string]interface{} `toml:"configs" yaml:"configs"`

	// Auths are registry endpoint to auth config mapping. The registry endpoint must
	// be a valid url with host specified.
	// DEPRECATED: Use Configs instead. Remove in containerd 1.4.
	Auths map[string]interface{} `toml:"auths" yaml:"auths"`
}

// Mirror contains the config related to the registry mirror
type Mirror struct {
	// Endpoints are endpoints for a namespace. CRI plugin will try the endpoints
	// one by one until a working one is found. The endpoint must be a valid url
	// with host specified.
	// The scheme, host and path from the endpoint URL will be used.
	Endpoints []string `toml:"endpoint" yaml:"endpoint"`
}

// getGlobalRegistriesConfFilename gets the global registries file that will be used in all the servers/workers
func getGlobalRegistriesConfFilename() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		log.Error("Couldn't get user's home directory")
		return "", err
	}

	return path.Join(homeDir, ".k3d", "registries.yaml"), nil
}

// writeRegistriesConfigInContainer creates a valid registries configuration file in a container
func writeRegistriesConfigInContainer(spec *ClusterSpec, ID string) error {
	registryInternalAddress := fmt.Sprintf("%s:%d", spec.RegistryName, defaultRegistryPort)
	registryExternalAddress := fmt.Sprintf("%s:%d", spec.RegistryName, spec.RegistryPort)

	privRegistries := &Registry{}

	// load the base registry file
	if len(spec.RegistriesFile) > 0 {
		log.Printf("Using registries definitions from %q...\n", spec.RegistriesFile)
		privRegistryFile, err := ioutil.ReadFile(spec.RegistriesFile)
		if err != nil {
			return err // the file must exist at this point
		}
		if err := yaml.Unmarshal(privRegistryFile, &privRegistries); err != nil {
			return err
		}
	}

	if spec.RegistryEnabled {
		if len(privRegistries.Mirrors) == 0 {
			privRegistries.Mirrors = map[string]Mirror{}
		}

		// the add the private registry
		privRegistries.Mirrors[registryExternalAddress] = Mirror{
			Endpoints: []string{fmt.Sprintf("http://%s", registryInternalAddress)},
		}
	}

	d, err := yaml.Marshal(&privRegistries)
	if err != nil {
		return err
	}

	return copyToContainer(ID, defaultFullRegistriesPath, d)
}

// createRegistry creates a registry, or connect the k3d network to an existing one
func createRegistry(spec ClusterSpec) (string, error) {
	netName := k3dNetworkName(spec.ClusterName)

	// first, check we have not already started a registry (for example, for a different k3d cluster)
	// all the k3d clusters should share the same private registry, so if we already have a registry just connect
	// it to the network of this cluster.
	cid, err := getRegistryContainer()
	if err != nil {
		return "", err
	}

	if cid != "" {
		// TODO: we should check given-registry-name == existing-registry-name
		log.Printf("Registry already present: connecting it to the %s network...\n", netName)
		if err := connectRegistryToNetwork(cid, netName, []string{spec.RegistryName}); err != nil {
			return "", err
		}
		return cid, nil
	}

	log.Printf("Creating Registry as %s:%d...\n", spec.RegistryName, spec.RegistryPort)

	containerLabels := make(map[string]string)

	// add a standard list of labels to our registry
	for k, v := range defaultRegistryLabels {
		containerLabels[k] = v
	}
	containerLabels["created"] = time.Now().Format("2006-01-02 15:04:05")
	containerLabels["hostname"] = spec.RegistryName

	registryPortSpec := fmt.Sprintf("0.0.0.0:%d:%d/tcp", spec.RegistryPort, defaultRegistryPort)
	registryPublishedPorts, err := CreatePublishedPorts([]string{registryPortSpec})
	if err != nil {
		log.Fatalf("Error: failed to parse port specs %+v \n%+v", registryPortSpec, err)
	}

	hostConfig := &container.HostConfig{
		PortBindings: registryPublishedPorts.PortBindings,
		Privileged:   true,
		Init:         &[]bool{true}[0],
	}

	if spec.AutoRestart {
		hostConfig.RestartPolicy.Name = "unless-stopped"
	}

	spec.Volumes = &Volumes{} // we do not need in the registry any of the volumes used by the other containers

	// connect the registry to this k3d network
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			netName: {
				Aliases: []string{spec.RegistryName},
			},
		},
	}

	config := &container.Config{
		Hostname:     spec.RegistryName,
		Image:        defaultRegistryImage,
		ExposedPorts: registryPublishedPorts.ExposedPorts,
		Labels:       containerLabels,
	}

	id, err := createContainer(config, hostConfig, networkingConfig, defaultRegistryContainerName)
	if err != nil {
		return "", fmt.Errorf(" Couldn't create registry container %s\n%w", defaultRegistryContainerName, err)
	}

	if err := startContainer(id); err != nil {
		return "", fmt.Errorf(" Couldn't start container %s\n%w", defaultRegistryContainerName, err)
	}

	return id, nil
}

// getRegistryContainer looks for the registry container
func getRegistryContainer() (string, error) {
	ctx := context.Background()
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("Couldn't create docker client\n%+v", err)
	}

	cFilter := filters.NewArgs()
	cFilter.Add("name", defaultRegistryContainerName)
	// filter with the standard list of labels of our registry
	for k, v := range defaultRegistryLabels {
		cFilter.Add("label", fmt.Sprintf("%s=%s", k, v))
	}

	containers, err := docker.ContainerList(ctx, types.ContainerListOptions{Filters: cFilter})
	if err != nil {
		return "", fmt.Errorf(" Couldn't list containers: %w", err)
	}
	if len(containers) == 0 {
		return "", nil
	}
	return containers[0].ID, nil
}

// connectRegistryToNetwork connects the registry container to a given network
func connectRegistryToNetwork(ID string, networkID string, aliases []string) error {
	if err := connectContainerToNetwork(ID, networkID, aliases); err != nil {
		return err
	}
	return nil
}

// disconnectRegistryFromNetwork disconnects the Registry from a Network
// if the Registry container is not connected to any more networks, it is stopped
func disconnectRegistryFromNetwork(name string) error {
	// disconnect the registry from this cluster's network
	netName := k3dNetworkName(name)
	cid, err := getRegistryContainer()
	if err != nil {
		return err
	}
	if cid == "" {
		return nil
	}

	log.Printf("...Disconnecting Registry from the %s network\n", netName)
	if err := disconnectContainerFromNetwork(cid, netName); err != nil {
		return err
	}

	// check if the registry is not connected to any other networks.
	// in that case, we can safely stop the registry container
	networks, err := getContainerNetworks(cid)
	if err != nil {
		return err
	}
	if len(networks) == 0 {
		log.Printf("...Removing the Registry\n")
		if err := removeContainer(cid); err != nil {
			log.Println(err)
		}
	}

	return nil
}
