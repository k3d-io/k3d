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
	k3dContainerd "github.com/rancher/k3d/pkg/runtimes/containerd"
	k3dDocker "github.com/rancher/k3d/pkg/runtimes/docker"
	k3d "github.com/rancher/k3d/pkg/types"
	log "github.com/sirupsen/logrus"
)

// CreateNodes creates a list of nodes
func CreateNodes(nodes []*k3d.Node, runtime k3drt.Runtime) { // TODO: pass `--atomic` flag, so we stop and return an error if any node creation fails?
	for _, node := range nodes {
		if err := CreateNode(node, runtime); err != nil {
			log.Error(err)
		}
	}
}

// CreateNode creates a new containerized k3s node
func CreateNode(node *k3d.Node, runtime k3drt.Runtime) error {
	log.Debugf("Creating node from spec\n%+v", node)

	/*
	 * CONFIGURATION
	 */

	// global node configuration (applies for any node role)
	labels := make(map[string]string)
	for k, v := range k3d.DefaultObjectLabels {
		labels[k] = v
	}
	for k, v := range node.Labels {
		labels[k] = v
	}
	node.Labels = labels

	// specify options depending on node role
	if node.Role == "worker" { // TODO: check here AND in CLI or only here?
		if err := patchWorkerSpec(node); err != nil {
			return err
		}
	} else if node.Role == "master" {
		if err := patchMasterSpec(node); err != nil {
			return err
		}
		log.Debugf("spec = %+v\n", node)
	} else {
		return fmt.Errorf("Unknown node role '%s'", node.Role)
	}

	/*
	 * CREATION
	 */
	if err := runtime.CreateNode(node); err != nil {
		return err
	}

	log.Debugln("...success")
	return nil
}

// DeleteNode deletes an existing node
func DeleteNode(node *k3d.Node, runtimeChoice string) error {
	var runtime k3drt.Runtime
	if runtimeChoice == "docker" {
		runtime = k3dDocker.Docker{}
	} else {
		runtime = k3dContainerd.Containerd{}
	}

	if err := runtime.DeleteNode(node); err != nil {
		log.Error(err)
	}
	log.Infoln("Deleted", node.Name)
	log.Debugln("...success")
	return nil
}

// patchWorkerSpec adds worker node specific settings to a node
func patchWorkerSpec(node *k3d.Node) error {
	node.Args = append([]string{"agent"}, node.Args...)
	node.Labels["role"] = "worker" // TODO: maybe put those in a global var DefaultWorkerNodeSpec?
	return nil
}

// patchMasterSpec adds worker node specific settings to a node
func patchMasterSpec(node *k3d.Node) error {
	node.Args = append([]string{"server"}, node.Args...)
	node.Labels["role"] = "master" // TODO: maybe put those in a global var DefaultMasterNodeSpec?
	return nil
}
