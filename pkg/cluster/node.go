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
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/imdario/mergo"
	"github.com/rancher/k3d/v3/pkg/runtimes"
	k3d "github.com/rancher/k3d/v3/pkg/types"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// NodeAddToCluster adds a node to an existing cluster
func NodeAddToCluster(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, cluster *k3d.Cluster, createNodeOpts k3d.NodeCreateOpts) error {
	targetClusterName := cluster.Name
	cluster, err := ClusterGet(ctx, runtime, cluster)
	if err != nil {
		log.Errorf("Failed to find specified cluster '%s'", targetClusterName)
		return err
	}

	// network
	node.Network = cluster.Network.Name

	// skeleton
	if node.Labels == nil {
		node.Labels = map[string]string{
			k3d.LabelRole: string(node.Role),
		}
	}
	node.Env = []string{}

	// copy labels and env vars from a similar node in the selected cluster
	var chosenNode *k3d.Node
	for _, existingNode := range cluster.Nodes {
		if existingNode.Role == node.Role {
			chosenNode = existingNode
			break
		}
	}
	// if we didn't find a node with the same role in the cluster, just choose any other node
	if chosenNode == nil {
		log.Debugf("Didn't find node with role '%s' in cluster '%s'. Choosing any other node (and using defaults)...", node.Role, cluster.Name)
		node.Cmd = k3d.DefaultRoleCmds[node.Role]
		for _, existingNode := range cluster.Nodes {
			if existingNode.Role != k3d.LoadBalancerRole { // any role except for the LoadBalancer role
				chosenNode = existingNode
				break
			}
		}
	}

	// get node details
	chosenNode, err = NodeGet(ctx, runtime, chosenNode)
	if err != nil {
		return err
	}

	log.Debugf("Adding node %+v \n>>> to cluster %+v\n>>> based on existing node %+v", node, cluster, chosenNode)

	// merge node config of new node into existing node config
	if err := mergo.MergeWithOverwrite(chosenNode, *node); err != nil {
		log.Errorln("Failed to merge new node config into existing node config")
		return err
	}

	node = chosenNode

	log.Debugf("Resulting node %+v", node)

	k3sURLFound := false
	for _, envVar := range node.Env {
		if strings.HasPrefix(envVar, "K3S_URL") {
			k3sURLFound = true
			break
		}
	}
	if !k3sURLFound {
		if url, ok := node.Labels[k3d.LabelClusterURL]; ok {
			node.Env = append(node.Env, fmt.Sprintf("K3S_URL=%s", url))
		} else {
			log.Warnln("Failed to find K3S_URL value!")
		}
	}

	if node.Role == k3d.ServerRole {
		for _, forbiddenCmd := range k3d.DoNotCopyServerFlags {
			for i, cmd := range node.Cmd {
				// cut out the '--cluster-init' flag as this should only be done by the initializing server node
				if cmd == forbiddenCmd {
					log.Debugf("Dropping '%s' from node's cmd", forbiddenCmd)
					node.Cmd = append(node.Cmd[:i], node.Cmd[i+1:]...)
				}
			}
			for i, arg := range node.Args {
				// cut out the '--cluster-init' flag as this should only be done by the initializing server node
				if arg == forbiddenCmd {
					log.Debugf("Dropping '%s' from node's args", forbiddenCmd)
					node.Args = append(node.Args[:i], node.Args[i+1:]...)
				}
			}
		}
	}

	if err := NodeCreate(ctx, runtime, node, k3d.NodeCreateOpts{}); err != nil {
		return err
	}

	// if it's a server node, then update the loadbalancer configuration
	if node.Role == k3d.ServerRole {
		if err := UpdateLoadbalancerConfig(ctx, runtime, cluster); err != nil {
			log.Errorln("Failed to update cluster loadbalancer")
			return err
		}
	}

	return nil
}

// NodeAddToClusterMulti adds multiple nodes to a chosen cluster
func NodeAddToClusterMulti(ctx context.Context, runtime runtimes.Runtime, nodes []*k3d.Node, cluster *k3d.Cluster, createNodeOpts k3d.NodeCreateOpts) error {
	if createNodeOpts.Timeout > 0*time.Second {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, createNodeOpts.Timeout)
		defer cancel()
	}

	nodeWaitGroup, ctx := errgroup.WithContext(ctx)
	for _, node := range nodes {
		if err := NodeAddToCluster(ctx, runtime, node, cluster, k3d.NodeCreateOpts{}); err != nil {
			return err
		}
		if createNodeOpts.Wait {
			currentNode := node
			nodeWaitGroup.Go(func() error {
				log.Debugf("Starting to wait for node '%s'", currentNode.Name)
				return NodeWaitForLogMessage(ctx, runtime, currentNode, k3d.ReadyLogMessageByRole[currentNode.Role], time.Time{})
			})
		}
	}
	if err := nodeWaitGroup.Wait(); err != nil {
		log.Errorln("Failed to bring up all nodes in time. Check the logs:")
		log.Errorf(">>> %+v", err)
		return fmt.Errorf("Failed to add nodes")
	}

	return nil
}

// NodeCreateMulti creates a list of nodes
func NodeCreateMulti(ctx context.Context, runtime runtimes.Runtime, nodes []*k3d.Node, createNodeOpts k3d.NodeCreateOpts) error { // TODO: pass `--atomic` flag, so we stop and return an error if any node creation fails?
	if createNodeOpts.Timeout > 0*time.Second {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, createNodeOpts.Timeout)
		defer cancel()
	}

	nodeWaitGroup, ctx := errgroup.WithContext(ctx)
	for _, node := range nodes {
		if err := NodeCreate(ctx, runtime, node, k3d.NodeCreateOpts{}); err != nil {
			log.Error(err)
		}
		if createNodeOpts.Wait {
			currentNode := node
			nodeWaitGroup.Go(func() error {
				log.Debugf("Starting to wait for node '%s'", currentNode.Name)
				return NodeWaitForLogMessage(ctx, runtime, currentNode, k3d.ReadyLogMessageByRole[currentNode.Role], time.Time{})
			})
		}
	}

	if err := nodeWaitGroup.Wait(); err != nil {
		log.Errorln("Failed to bring up all nodes in time. Check the logs:")
		log.Errorf(">>> %+v", err)
		return fmt.Errorf("Failed to create nodes")
	}

	return nil

}

// NodeCreate creates a new containerized k3s node
func NodeCreate(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, createNodeOpts k3d.NodeCreateOpts) error {
	log.Debugf("Creating node from spec\n%+v", node)

	/*
	 * CONFIGURATION
	 */

	/* global node configuration (applies for any node role) */

	// ### Labels ###
	labels := make(map[string]string)
	for k, v := range k3d.DefaultObjectLabels {
		labels[k] = v
	}
	for k, v := range node.Labels {
		labels[k] = v
	}
	node.Labels = labels
	// second most important: the node role label
	node.Labels[k3d.LabelRole] = string(node.Role)

	// ### Environment ###
	node.Env = append(node.Env, k3d.DefaultNodeEnv...) // append default node env vars

	// specify options depending on node role
	if node.Role == k3d.AgentRole { // TODO: check here AND in CLI or only here?
		if err := patchAgentSpec(node); err != nil {
			return err
		}
	} else if node.Role == k3d.ServerRole {
		if err := patchServerSpec(node); err != nil {
			return err
		}
	}

	/*
	 * CREATION
	 */
	if err := runtime.CreateNode(ctx, node); err != nil {
		return err
	}

	return nil
}

// NodeDelete deletes an existing node
func NodeDelete(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node) error {

	if err := runtime.DeleteNode(ctx, node); err != nil {
		log.Error(err)
	}

	cluster, err := ClusterGet(ctx, runtime, &k3d.Cluster{Name: node.Labels[k3d.LabelClusterName]})
	if err != nil {
		log.Errorf("Failed to update loadbalancer: Failed to find cluster for node '%s'", node.Name)
		return err
	}

	// if it's a server node, then update the loadbalancer configuration
	if node.Role == k3d.ServerRole {
		if err := UpdateLoadbalancerConfig(ctx, runtime, cluster); err != nil {
			log.Errorln("Failed to update cluster loadbalancer")
			return err
		}
	}

	return nil
}

// patchAgentSpec adds agent node specific settings to a node
func patchAgentSpec(node *k3d.Node) error {
	if node.Cmd == nil {
		node.Cmd = []string{"agent"}
	}
	return nil
}

// patchServerSpec adds agent node specific settings to a node
func patchServerSpec(node *k3d.Node) error {

	// command / arguments
	if node.Cmd == nil {
		node.Cmd = []string{"server"}
	}

	// Add labels and TLS SAN for the exposed API
	// FIXME: For now, the labels concerning the API on the server nodes are only being used for configuring the kubeconfig
	node.Labels[k3d.LabelServerAPIHostIP] = node.ServerOpts.ExposeAPI.HostIP // TODO: maybe get docker machine IP here
	node.Labels[k3d.LabelServerAPIHost] = node.ServerOpts.ExposeAPI.Host
	node.Labels[k3d.LabelServerAPIPort] = node.ServerOpts.ExposeAPI.Port

	node.Args = append(node.Args, "--tls-san", node.ServerOpts.ExposeAPI.Host) // add TLS SAN for non default host name

	return nil
}

// NodeList returns a list of all existing clusters
func NodeList(ctx context.Context, runtime runtimes.Runtime) ([]*k3d.Node, error) {
	nodes, err := runtime.GetNodesByLabel(ctx, k3d.DefaultObjectLabels)
	if err != nil {
		log.Errorln("Failed to get nodes")
		return nil, err
	}

	return nodes, nil
}

// NodeGet returns a node matching the specified node fields
func NodeGet(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node) (*k3d.Node, error) {
	// get node
	node, err := runtime.GetNode(ctx, node)
	if err != nil {
		log.Errorf("Failed to get node '%s'", node.Name)
	}

	return node, nil
}

// NodeWaitForLogMessage follows the logs of a node container and returns if it finds a specific line in there (or timeout is reached)
func NodeWaitForLogMessage(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, message string, since time.Time) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// read the logs
		out, err := runtime.GetNodeLogs(ctx, node, since)
		if err != nil {
			if out != nil {
				out.Close()
			}
			log.Errorf("Failed waiting for log message '%s' from node '%s'", message, node.Name)
			return err
		}
		defer out.Close()

		buf := new(bytes.Buffer)
		nRead, _ := buf.ReadFrom(out)
		out.Close()
		output := buf.String()

		// check if we can find the specified line in the log
		if nRead > 0 && strings.Contains(output, message) {
			break
		}
	}
	time.Sleep(500 * time.Millisecond) // wait for half a second to avoid overloading docker (error `socket: too many open files`)
	log.Debugf("Finished waiting for log message '%s' from node '%s'", message, node.Name)
	return nil
}
