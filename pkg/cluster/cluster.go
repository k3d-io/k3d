/*
Copyright © 2019 Thorsten Klein <iwilltry42@gmail.com>

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
	"fmt"

	k3drt "github.com/rancher/k3d/pkg/runtimes"
	k3d "github.com/rancher/k3d/pkg/types"
	"github.com/rancher/k3d/pkg/util"
	log "github.com/sirupsen/logrus"
)

// CreateCluster creates a new cluster consisting of
// - some containerized k3s nodes
// - a docker network
func CreateCluster(cluster *k3d.Cluster, runtime k3drt.Runtime) error {

	/*
	 * Network
	 */

	// generate cluster network name, if not set
	if cluster.Network == "" {
		cluster.Network = fmt.Sprintf("%s-%s", k3d.DefaultObjectNamePrefix, cluster.Name)
	}

	// create cluster network or use an existing one
	networkID, err := runtime.CreateNetworkIfNotPresent(cluster.Network)
	if err != nil {
		log.Errorln("Failed to create cluster network")
		return err
	}
	cluster.Network = networkID

	/*
	 * Cluster Secret
	 */

	if cluster.Secret == "" {
		cluster.Secret = GenerateClusterSecret()
	}

	/*
	 * Nodes
	 */

	// used for node suffices
	masterCount := 0
	workerCount := 0

	for _, node := range cluster.Nodes {

		// cluster specific settings
		node.Labels = make(map[string]string)
		node.Labels["cluster"] = cluster.Name
		node.Env = append(node.Env, fmt.Sprintf("K3S_CLUSTER_SECRET=%s", cluster.Secret))

		// node role specific settings
		suffix := 0
		if node.Role == k3d.MasterRole {
			// name suffix
			suffix = masterCount
			masterCount++
		} else if node.Role == k3d.WorkerRole {
			// name suffix
			suffix = workerCount
			workerCount++

			// connection url
			node.Env = append(node.Env, fmt.Sprintf("K3S_URL=https://%s:%d", generateNodeName(cluster.Name, k3d.MasterRole, 0), 6443)) // TODO: use actual configurable api-port
		}

		node.Name = generateNodeName(cluster.Name, node.Role, suffix)
		node.Network = cluster.Network

		// create node
		log.Infof("Creating node '%s'", node.Name)
		if err := CreateNode(&node, runtime); err != nil {
			log.Errorln("Failed to create node")
			return err
		}
		log.Debugf("Created node '%s'", node.Name)
	}

	return nil
}

// DeleteCluster deletes an existing cluster
func DeleteCluster(cluster *k3d.Cluster, runtime k3drt.Runtime) error {
	return nil
}

// GetClusters returns a list of all existing clusters
func GetClusters(runtime k3drt.Runtime) ([]*k3d.Cluster, error) {
	runtime.GetNodesByLabel(map[string]string{"role": string(k3d.MasterRole)})
	return []*k3d.Cluster{}, nil
}

// GetCluster returns an existing cluster
func GetCluster(cluster *k3d.Cluster, runtime k3drt.Runtime) (*k3d.Cluster, error) {
	return cluster, nil
}

// GenerateClusterSecret generates a random 20 character string
func GenerateClusterSecret() string {
	return util.GenerateRandomString(20)
}

func generateNodeName(cluster string, role k3d.Role, suffix int) string {
	return fmt.Sprintf("%s-%s-%s-%d", k3d.DefaultObjectNamePrefix, cluster, role, suffix)
}
