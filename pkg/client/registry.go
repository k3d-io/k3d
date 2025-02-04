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
package client

import (
	"context"
	"fmt"
	gort "runtime"

	wharfie "github.com/rancher/wharfie/pkg/registries"

	"github.com/docker/go-connections/nat"
	"github.com/imdario/mergo"
	"sigs.k8s.io/yaml"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	"github.com/k3d-io/k3d/v5/pkg/runtimes/docker"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/pkg/types/k8s"
)

func RegistryRun(ctx context.Context, runtime runtimes.Runtime, reg *k3d.Registry) (*k3d.Node, error) {
	regNode, err := RegistryCreate(ctx, runtime, reg)
	if err != nil {
		return nil, fmt.Errorf("Failed to create registry: %+v", err)
	}

	if err := NodeStart(ctx, runtime, regNode, &k3d.NodeStartOpts{}); err != nil {
		return nil, fmt.Errorf("Failed to start registry: %+v", err)
	}

	return regNode, nil
}

// RegistryCreate creates a registry node
func RegistryCreate(ctx context.Context, runtime runtimes.Runtime, reg *k3d.Registry) (*k3d.Node, error) {
	// registry name
	if len(reg.Host) == 0 {
		reg.Host = k3d.DefaultRegistryName
	}
	// if err := ValidateHostname(reg.Host); err != nil {
	// 	l.Log().Errorln("Invalid name for registry")
	// 	l.Log().Fatalln(err)
	// }

	if len(reg.Network) == 0 {
		reg.Network = k3d.DefaultRuntimeNetwork
	}

	registryNode := &k3d.Node{
		Name:     reg.Host,
		Image:    reg.Image,
		Role:     k3d.RegistryRole,
		Networks: []string{reg.Network},
		Restart:  true,
		Env:      []string{},
	}

	if reg.ExposureOpts.Binding.HostPort != "" {
		registryNode.Env = append(registryNode.Env, fmt.Sprintf("REGISTRY_HTTP_ADDR=:%s", reg.ExposureOpts.Binding.HostPort))
	}

	if reg.Options.Proxy.RemoteURL != "" {
		registryNode.Env = append(registryNode.Env, fmt.Sprintf("REGISTRY_PROXY_REMOTEURL=%s", reg.Options.Proxy.RemoteURL))

		if reg.Options.Proxy.Username != "" {
			registryNode.Env = append(registryNode.Env, fmt.Sprintf("REGISTRY_PROXY_USERNAME=%s", reg.Options.Proxy.Username))
		}

		if reg.Options.Proxy.Password != "" {
			registryNode.Env = append(registryNode.Env, fmt.Sprintf("REGISTRY_PROXY_PASSWORD=%s", reg.Options.Proxy.Password))
		}
	}

	if reg.Options.DeleteEnabled {
		registryNode.Env = append(registryNode.Env, "REGISTRY_STORAGE_DELETE_ENABLED=true")
	}

	if len(reg.Volumes) > 0 {
		registryNode.Volumes = reg.Volumes
	}

	// error out if that registry exists already
	existingNode, err := runtime.GetNode(ctx, registryNode)
	if err == nil && existingNode != nil {
		return nil, fmt.Errorf("A registry node with that name already exists")
	}

	// setup the node labels
	registryNode.RuntimeLabels = map[string]string{
		k3d.LabelClusterName:          reg.ClusterRef,
		k3d.LabelRole:                 string(k3d.RegistryRole),
		k3d.LabelRegistryHost:         reg.ExposureOpts.Host, // TODO: docker machine host?
		k3d.LabelRegistryHostIP:       reg.ExposureOpts.Binding.HostIP,
		k3d.LabelRegistryPortExternal: reg.ExposureOpts.Binding.HostPort,
		k3d.LabelRegistryPortInternal: reg.ExposureOpts.Port.Port(),
	}
	for k, v := range k3d.DefaultRuntimeLabels {
		registryNode.RuntimeLabels[k] = v
	}
	for k, v := range k3d.DefaultRuntimeLabelsVar {
		registryNode.RuntimeLabels[k] = v
	}

	// port
	registryNode.Ports = nat.PortMap{}
	registryNode.Ports[reg.ExposureOpts.Port] = []nat.PortBinding{reg.ExposureOpts.Binding}

	// create the registry node
	l.Log().Infof("Creating node '%s'", registryNode.Name)
	if err := NodeCreate(ctx, runtime, registryNode, k3d.NodeCreateOpts{}); err != nil {
		return nil, fmt.Errorf("failed to create registry node '%s': %w", registryNode.Name, err)
	}

	l.Log().Infof("Successfully created registry '%s'", registryNode.Name)

	return registryNode, nil
}

// RegistryConnectClusters connects an existing registry to one or more clusters
func RegistryConnectClusters(ctx context.Context, runtime runtimes.Runtime, registryNode *k3d.Node, clusters []*k3d.Cluster) error {
	// find registry node
	registryNode, err := NodeGet(ctx, runtime, registryNode)
	if err != nil {
		return fmt.Errorf("Failed to find registry node '%s': %w", registryNode.Name, err)
	}

	// get cluster details and connect
	failed := 0
	for _, c := range clusters {
		cluster, err := ClusterGet(ctx, runtime, c)
		if err != nil {
			l.Log().Warnf("Failed to connect to cluster '%s': Cluster not found", c.Name)
			failed++
			continue
		}
		if err := runtime.ConnectNodeToNetwork(ctx, registryNode, cluster.Network.Name); err != nil {
			l.Log().Warnf("Failed to connect to cluster '%s': Connection failed", cluster.Name)
			l.Log().Warnln(err)
			failed++
		}
	}

	if failed > 0 {
		return fmt.Errorf("Failed to connect to one or more clusters")
	}

	return nil
}

// RegistryConnectNetworks connects an existing registry to one or more networks
func RegistryConnectNetworks(ctx context.Context, runtime runtimes.Runtime, registryNode *k3d.Node, networks []string) error {
	// find registry node
	registryNode, err := NodeGet(ctx, runtime, registryNode)
	if err != nil {
		return fmt.Errorf("Failed to find registry node '%s': %w", registryNode.Name, err)
	}

	// get cluster details and connect
	failed := 0
	for _, net := range networks {
		if err := runtime.ConnectNodeToNetwork(ctx, registryNode, net); err != nil {
			l.Log().Warnf("Failed to connect to network '%s': Connection failed", net)
			l.Log().Warnln(err)
			failed++
		}
	}

	if failed > 0 {
		return fmt.Errorf("Failed to connect to one or more networks")
	}

	return nil
}

// RegistryGenerateK3sConfig generates the k3s specific registries.yaml configuration for multiple registries
func RegistryGenerateK3sConfig(ctx context.Context, registries []*k3d.Registry) (*wharfie.Registry, error) {
	regConf := &wharfie.Registry{}
	rewritesConf := make(map[string]string)

	for _, reg := range registries {
		internalAddress := fmt.Sprintf("%s:%s", reg.Host, reg.ExposureOpts.Port.Port())
		externalAddress := fmt.Sprintf("%s:%s", reg.Host, reg.ExposureOpts.Binding.HostPort)

		// init mirrors if nil
		if regConf.Mirrors == nil {
			regConf.Mirrors = make(map[string]wharfie.Mirror)
		}

		regConf.Mirrors[externalAddress] = wharfie.Mirror{
			Endpoints: []string{
				fmt.Sprintf("http://%s", internalAddress),
			},
			Rewrites: rewritesConf,
		}

		regConf.Mirrors[internalAddress] = wharfie.Mirror{
			Endpoints: []string{
				fmt.Sprintf("http://%s", internalAddress),
			},
			Rewrites: rewritesConf, // stub out rewrites so we dont override with nil
		}

		if reg.Options.Proxy.RemoteURL != "" {
			regConf.Mirrors[reg.Options.Proxy.RemoteURL] = wharfie.Mirror{
				Endpoints: []string{fmt.Sprintf("http://%s", internalAddress)},
			}
		}
	}

	return regConf, nil
}

// RegistryGet gets a registry node by name and returns it as a registry object
func RegistryGet(ctx context.Context, runtime runtimes.Runtime, name string) (*k3d.Registry, error) {
	regNode, err := runtime.GetNode(ctx, &k3d.Node{
		Name: name,
		Role: k3d.RegistryRole,
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to find registry '%s': %+v", name, err)
	}

	registry := &k3d.Registry{
		Host: regNode.Name,
	}
	// TODO: finish RegistryGet
	return registry, nil
}

// RegistryFromNode transforms a node spec to a registry spec
func RegistryFromNode(node *k3d.Node) (*k3d.Registry, error) {
	registry := &k3d.Registry{
		Host:  node.Name,
		Image: node.Image,
	}

	// we expect exactly one portmap
	if len(node.Ports) != 1 {
		return nil, fmt.Errorf("Failed to parse registry spec from node %+v: 0 or multiple ports defined, where one is expected", node)
	}

	for port, bindings := range node.Ports {
		registry.ExposureOpts.Port = port

		// we expect 0 or 1 binding for that port
		if len(bindings) > 1 {
			return nil, fmt.Errorf("Failed to parse registry spec from node %+v: Multiple bindings '%+v' specified for port '%s' where one is expected", node, bindings, port)
		}

		for _, binding := range bindings {
			registry.ExposureOpts.Binding = binding
		}
	}

	l.Log().Tracef("Got registry %+v from node %+v", registry, node)

	return registry, nil
}

// RegistryGenerateLocalRegistryHostingConfigMapYAML generates a ConfigMap used to advertise the registries in the cluster
func RegistryGenerateLocalRegistryHostingConfigMapYAML(ctx context.Context, runtime runtimes.Runtime, registries []*k3d.Registry) ([]byte, error) {
	type cmMetadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}

	type cmData struct {
		RegHostV1 string `json:"localRegistryHosting.v1"`
	}

	type configmap struct {
		APIVersion string     `json:"apiVersion"`
		Kind       string     `json:"kind"`
		Metadata   cmMetadata `json:"metadata"`
		Data       cmData     `json:"data"`
	}

	if len(registries) > 1 {
		l.Log().Warnf("More than one registry specified, but the LocalRegistryHostingV1 spec only supports one -> Selecting the first one: %s", registries[0].Host)
	}

	if len(registries) < 1 {
		l.Log().Debugln("No registry specified, not generating local registry hosting configmap")
		return nil, nil
	}

	// if no host is set, fallback onto the HostIP used to bind the port
	host := registries[0].ExposureOpts.Host
	if host == "" {
		host = registries[0].ExposureOpts.Binding.HostIP
	}

	// if the host is now 0.0.0.0, check if we can set it to the IP of the docker-machine, if it's used
	if host == k3d.DefaultAPIHost && runtime == runtimes.Docker {
		if gort.GOOS == "windows" || gort.GOOS == "darwin" {
			l.Log().Tracef("Running on %s: checking if it's using docker-machine", gort.GOOS)
			machineIP, err := runtime.(docker.Docker).GetDockerMachineIP()
			if err != nil {
				l.Log().Warnf("Using docker-machine, but failed to get it's IP for usage in LocalRegistryHosting Config Map: %+v", err)
			} else if machineIP != "" {
				l.Log().Infof("Using the docker-machine IP %s in the LocalRegistryHosting Config Map", machineIP)
				host = machineIP
			} else {
				l.Log().Traceln("Not using docker-machine")
			}
		}
	}

	// if host is still 0.0.0.0, use localhost instead
	if host == k3d.DefaultAPIHost {
		host = "localhost" // we prefer localhost over 0.0.0.0
	}

	// transform configmap data to YAML
	dat, err := yaml.Marshal(
		k8s.LocalRegistryHostingV1{
			Host:                     fmt.Sprintf("%s:%s", host, registries[0].ExposureOpts.Binding.HostPort),
			HostFromContainerRuntime: fmt.Sprintf("%s:%s", registries[0].Host, registries[0].ExposureOpts.Port.Port()),
			HostFromClusterNetwork:   fmt.Sprintf("%s:%s", registries[0].Host, registries[0].ExposureOpts.Port.Port()),
			Help:                     "https://k3d.io/stable/usage/registries/#using-a-local-registry",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal LocalRegistryHosting configmap data: %w", err)
	}

	cm := configmap{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Metadata: cmMetadata{
			Name:      "local-registry-hosting",
			Namespace: "kube-public",
		},
		Data: cmData{
			RegHostV1: string(dat),
		},
	}

	cmYaml, err := yaml.Marshal(cm)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal LocalRegistryHosting configmap: %w", err)
	}

	l.Log().Tracef("LocalRegistryHostingConfigMapYaml: %s", string(cmYaml))

	return cmYaml, nil
}

// RegistryMergeConfig merges a source registry config into an existing dest registry config
func RegistryMergeConfig(ctx context.Context, dest, src *wharfie.Registry) error {
	if err := mergo.MergeWithOverwrite(dest, src); err != nil {
		return fmt.Errorf("failed to merge registry configs: %w", err)
	}
	return nil
}
