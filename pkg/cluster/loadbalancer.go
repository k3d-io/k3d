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
package cluster

import (
	"context"
	"fmt"
	"strings"

	"github.com/rancher/k3d/pkg/runtimes"
	k3d "github.com/rancher/k3d/pkg/types"
	log "github.com/sirupsen/logrus"
)

// UpdateLoadbalancerConfig updates the loadbalancer config with an updated list of servers belonging to that cluster
func UpdateLoadbalancerConfig(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster) error {

	var err error
	// update cluster details to ensure that we have the latest node list
	cluster, err = GetCluster(ctx, runtime, cluster)
	if err != nil {
		log.Errorln("Failed to update details for cluster '%s'", cluster.Name)
		return err
	}

	// find the LoadBalancer for the target cluster
	masterNodesList := []string{}
	var loadbalancer *k3d.Node
	for _, node := range cluster.Nodes {
		if node.Role == k3d.LoadBalancerRole { // get the loadbalancer we want to update
			loadbalancer = node
		} else if node.Role == k3d.MasterRole { // create a list of master nodes
			masterNodesList = append(masterNodesList, node.Name)
		}
	}
	masterNodes := strings.Join(masterNodesList, ",")
	if loadbalancer == nil {
		return fmt.Errorf("Failed to find loadbalancer for cluster '%s'", cluster.Name)
	}

	log.Debugf("Servers as passed to masterlb: '%s'", masterNodes)

	command := fmt.Sprintf("SERVERS=%s %s", masterNodes, "confd -onetime -backend env && nginx -s reload")
	if err := runtime.ExecInNode(ctx, loadbalancer, []string{"sh", "-c", command}); err != nil {
		log.Errorln("Failed to update loadbalancer configuration")
		return err
	}

	return nil
}
