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

	"github.com/rancher/k3d/v4/pkg/runtimes"
	"github.com/rancher/k3d/v4/pkg/types"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
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

	// find the LoadBalancer for the target cluster
	serverNodesList := []string{}
	var loadbalancer *k3d.Node
	for _, node := range cluster.Nodes {
		if node.Role == k3d.LoadBalancerRole { // get the loadbalancer we want to update
			loadbalancer = node
		} else if node.Role == k3d.ServerRole { // create a list of server nodes
			serverNodesList = append(serverNodesList, node.Name)
		}
	}
	serverNodes := strings.Join(serverNodesList, ",")
	if loadbalancer == nil {
		return fmt.Errorf("Failed to find loadbalancer for cluster '%s'", cluster.Name)
	}

	log.Debugf("Servers as passed to serverlb: '%s'", serverNodes)

	command := fmt.Sprintf("SERVERS=%s %s", serverNodes, "confd -onetime -backend env && nginx -s reload")
	if err := runtime.ExecInNode(ctx, loadbalancer, []string{"sh", "-c", command}); err != nil {
		if strings.Contains(err.Error(), "host not found in upstream") {
			log.Warnf("Loadbalancer configuration updated, but one or more k3d nodes seem to be down, check the logs:\n%s", err.Error())
			return nil
		}
		return err
	}

	return nil
}

func GetLoadbalancerConfig(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster) (*types.LoadbalancerConfig, error) {

	if cluster.ServerLoadBalancer == nil {
		for _, node := range cluster.Nodes {
			if node.Role == types.LoadBalancerRole {
				var err error
				cluster.ServerLoadBalancer, err = NodeGet(ctx, runtime, node)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	reader, err := runtime.ReadFromNode(ctx, types.DefaultLoadbalancerConfigPath, cluster.ServerLoadBalancer)
	if err != nil {
		return &k3d.LoadbalancerConfig{}, err
	}
	defer reader.Close()

	file, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	file = bytes.Trim(file[512:], "\x00") // trim control characters, etc.

	currentConfig := &types.LoadbalancerConfig{}
	if err := yaml.Unmarshal(file, currentConfig); err != nil {
		return nil, err
	}

	return currentConfig, nil
}
