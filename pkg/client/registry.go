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
package client

import (
	"context"
	"fmt"

	"github.com/rancher/k3d/v4/pkg/runtimes"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	"github.com/rancher/k3d/v4/pkg/types/k3s"
	log "github.com/sirupsen/logrus"
)

// RegistryCreate creates a registry node
func RegistryCreate(ctx context.Context, runtime runtimes.Runtime, reg *k3d.Registry) (*k3d.Node, error) {

	// registry name
	if len(reg.Name) == 0 {
		reg.Name = k3d.DefaultRegistryName
	}
	if err := ValidateHostname(reg.Name); err != nil {
		log.Errorln("Invalid name for registry")
		log.Fatalln(err)
	}

	registryNode := &k3d.Node{
		Name:    reg.Name,
		Image:   reg.Image,
		Role:    k3d.RegistryRole,
		Network: "bridge", // Default network: TODO: change to const from types
	}

	// error out if that registry exists already
	existingNode, err := runtime.GetNode(ctx, registryNode)
	if err == nil && existingNode != nil {
		return nil, fmt.Errorf("A registry node with that name already exists")
	}

	// setup the node labels
	registryNode.Labels = map[string]string{
		k3d.LabelRole:           string(k3d.RegistryRole),
		k3d.LabelRegistryHost:   reg.Port.Host, // TODO: docker machine host?
		k3d.LabelRegistryHostIP: reg.Port.HostIP,
		k3d.LabelRegistryPort:   reg.Port.Port,
	}
	for k, v := range k3d.DefaultObjectLabels {
		registryNode.Labels[k] = v
	}

	// port
	registryNode.Ports = []string{
		fmt.Sprintf("%s:%s:%s/tcp", reg.Port.HostIP, reg.Port.Port, k3d.DefaultRegistryPort),
	}

	// create the registry node
	log.Infof("Creating node '%s'", registryNode.Name)
	if err := NodeCreate(ctx, runtime, registryNode, k3d.NodeCreateOpts{}); err != nil {
		log.Errorln("Failed to create registry node")
		return nil, err
	}

	log.Infof("Successfully created registry '%s'", registryNode.Name)

	return registryNode, nil

}

// RegistryConnect connects an existing registry to one or more clusters
func RegistryConnect(ctx context.Context, runtime runtimes.Runtime, registryNode *k3d.Node, clusters []*k3d.Cluster) error {

	// find registry node
	registryNode, err := NodeGet(ctx, runtime, registryNode)
	if err != nil {
		log.Errorf("Failed to find registry node '%s'", registryNode.Name)
		return err
	}

	// get cluster details and connect
	failed := 0
	for _, c := range clusters {
		cluster, err := ClusterGet(ctx, runtime, c)
		if err != nil {
			log.Warnf("Failed to connect to cluster '%s': Cluster not found", cluster.Name)
			failed++
			continue
		}
		if err := runtime.ConnectNodeToNetwork(ctx, registryNode, cluster.Network.Name); err != nil {
			log.Warnf("Failed to connect to cluster '%s': Connection failed", cluster.Name)
			log.Warnln(err)
			failed++
		}
	}

	if failed > 0 {
		return fmt.Errorf("Failed to connect to one or more clusters")
	}

	return nil
}

// RegistryGenerateK3sConfig generates the k3s specific registries.yaml configuration for multiple registries
func RegistryGenerateK3sConfig(ctx context.Context, internalRegistries *k3d.Registry, externalRegistries []*k3d.ExternalRegistry) (*k3s.Registry, error) {
	regConf := &k3s.Registry{}

	for _, reg := range externalRegistries {
		internalAddress := fmt.Sprintf("%s:%s", reg.Name, reg.Port)
		externalAddress := fmt.Sprintf("%s:%s", reg.Name, reg.ExternalPort)

		// init mirrors if nil
		if regConf.Mirrors == nil {
			regConf.Mirrors = make(map[string]k3s.Mirror)
		}

		regConf.Mirrors[externalAddress] = k3s.Mirror{
			Endpoints: []string{
				fmt.Sprintf("http://%s", internalAddress),
			},
		}

		if reg.Proxy != "" {
			regConf.Mirrors[k3d.DefaultDockerHubAddress] = k3s.Mirror{
				Endpoints: []string{fmt.Sprintf("http://%s", internalAddress)},
			}
		}
	}

	return regConf, nil
}
