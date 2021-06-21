/*
Copyright © 2020-2021 The k3d Author(s)

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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/docker/go-connections/nat"
	"github.com/go-test/deep"
	"github.com/rancher/k3d/v4/pkg/runtimes"
	"github.com/rancher/k3d/v4/pkg/types"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// UpdateLoadbalancerConfig updates the loadbalancer config with an updated list of servers belonging to that cluster
func UpdateLoadbalancerConfig(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster) error {

	var err error
	// update cluster details to ensure that we have the latest node list
	cluster, err = ClusterGet(ctx, runtime, cluster)
	if err != nil {
		log.Errorf("Failed to update details for cluster '%s'", cluster.Name)
		return err
	}

	currentConfig, err := GetLoadbalancerConfig(ctx, runtime, cluster)
	if err != nil {
		return fmt.Errorf("error getting current config from loadbalancer: %w", err)
	}

	log.Tracef("Current loadbalancer config:\n%+v", currentConfig)

	newLBConfig, err := LoadbalancerGenerateConfig(cluster)
	if err != nil {
		return fmt.Errorf("error generating new loadbalancer config: %w", err)
	}
	log.Tracef("New loadbalancer config:\n%+v", currentConfig)

	if diff := deep.Equal(currentConfig, newLBConfig); diff != nil {
		log.Debugf("Updating the loadbalancer with this diff: %+v", diff)
	}

	newLbConfigYaml, err := yaml.Marshal(&newLBConfig)
	if err != nil {
		return fmt.Errorf("error marshalling the new loadbalancer config: %w", err)
	}
	log.Debugf("Writing lb config:\n%s", string(newLbConfigYaml))
	if err := runtime.WriteToNode(ctx, newLbConfigYaml, k3d.DefaultLoadbalancerConfigPath, 0744, cluster.ServerLoadBalancer); err != nil {
		return fmt.Errorf("error writing new loadbalancer config to container: %w", err)
	}

	command := "confd -onetime -backend file -file /etc/confd/values.yaml -log-level debug && nginx -s reload"
	if err := runtime.ExecInNode(ctx, cluster.ServerLoadBalancer, []string{"sh", "-c", command}); err != nil {
		if strings.Contains(err.Error(), "host not found in upstream") {
			log.Warnf("Loadbalancer configuration updated, but one or more k3d nodes seem to be down, check the logs:\n%s", err.Error())
			return nil
		}
		return err
	}

	return nil
}

func GetLoadbalancerConfig(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster) (types.LoadbalancerConfig, error) {

	var cfg k3d.LoadbalancerConfig

	if cluster.ServerLoadBalancer == nil {
		for _, node := range cluster.Nodes {
			if node.Role == types.LoadBalancerRole {
				var err error
				cluster.ServerLoadBalancer, err = NodeGet(ctx, runtime, node)
				if err != nil {
					return cfg, err
				}
			}
		}
	}

	reader, err := runtime.ReadFromNode(ctx, types.DefaultLoadbalancerConfigPath, cluster.ServerLoadBalancer)
	if err != nil {
		return cfg, err
	}
	defer reader.Close()

	file, err := ioutil.ReadAll(reader)
	if err != nil {
		return cfg, err
	}

	file = bytes.Trim(file[512:], "\x00") // trim control characters, etc.

	if err := yaml.Unmarshal(file, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func LoadbalancerGenerateConfig(cluster *k3d.Cluster) (k3d.LoadbalancerConfig, error) {
	lbConfig := k3d.LoadbalancerConfig{
		Ports:    map[string][]string{},
		Settings: k3d.LoadBalancerSettings{},
	}

	// get list of server nodes
	servers := []string{}
	for _, node := range cluster.Nodes {
		if node.Role == k3d.ServerRole {
			servers = append(servers, node.Name)
		}
	}

	// Default API Port proxied to the server nodes
	lbConfig.Ports[fmt.Sprintf("%s.tcp", k3d.DefaultAPIPort)] = servers

	// generate comma-separated list of extra ports to forward // TODO: no default targets?
	for exposedPort := range cluster.ServerLoadBalancer.Ports {
		// TODO: catch duplicates here?
		lbConfig.Ports[fmt.Sprintf("%s.%s", exposedPort.Port(), exposedPort.Proto())] = servers
	}

	// some additional nginx settings
	lbConfig.Settings.WorkerProcesses = k3d.DefaultLoadbalancerWorkerProcesses + len(cluster.ServerLoadBalancer.Ports)*len(servers)

	return lbConfig, nil
}

func LoadbalancerPrepare(ctx context.Context, runtime runtimes.Runtime, cluster *types.Cluster, opts *k3d.LoadbalancerCreateOpts) (*k3d.Node, error) {

	if cluster.ServerLoadBalancer.Ports == nil {
		cluster.ServerLoadBalancer.Ports = nat.PortMap{}
	}
	cluster.ServerLoadBalancer.Ports[k3d.DefaultAPIPort] = []nat.PortBinding{cluster.KubeAPI.Binding}

	// Create LB as a modified node with loadbalancerRole
	lbNode := &k3d.Node{
		Name:          fmt.Sprintf("%s-%s-serverlb", k3d.DefaultObjectNamePrefix, cluster.Name),
		Image:         k3d.GetLoadbalancerImage(),
		Ports:         cluster.ServerLoadBalancer.Ports,
		Role:          k3d.LoadBalancerRole,
		RuntimeLabels: opts.Labels, // TODO: createLoadBalancer: add more expressive labels
		Networks:      []string{cluster.Network.Name},
		Restart:       true,
	}

	return lbNode, nil

}
