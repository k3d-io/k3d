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
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/rancher/k3d/pkg/runtimes"
	k3d "github.com/rancher/k3d/pkg/types"
	"github.com/rancher/k3d/pkg/util"
	log "github.com/sirupsen/logrus"
	kubeconfig "k8s.io/client-go/tools/clientcmd"
)

// GetKubeconfig grabs the kubeconfig file from /output from a master node container and puts it into a local directory
func GetKubeconfig(runtime runtimes.Runtime, cluster *k3d.Cluster) ([]byte, error) {
	// get all master nodes for the selected cluster
	// TODO: getKubeconfig: we should make sure, that the master node we're trying to getch is actually running
	masterNodes, err := runtime.GetNodesByLabel(map[string]string{"k3d.cluster": cluster.Name, "k3d.role": string(k3d.MasterRole)})
	if err != nil {
		log.Errorln("Failed to get master nodes")
		return nil, err
	}
	if len(masterNodes) == 0 {
		return nil, fmt.Errorf("Didn't find any master node")
	}

	// prefer a master node, which actually has the port exposed
	var chosenMaster *k3d.Node
	chosenMaster = nil
	APIPort := k3d.DefaultAPIPort
	APIHost := k3d.DefaultAPIHost

	for _, master := range masterNodes {
		if _, ok := master.Labels["k3d.master.api.port"]; ok {
			chosenMaster = master
			APIPort = master.Labels["k3d.master.api.port"]
			if _, ok := master.Labels["k3d.master.api.host"]; ok {
				APIHost = master.Labels["k3d.master.api.host"]
			}
			break
		}
	}

	if chosenMaster == nil {
		chosenMaster = masterNodes[0]
	}

	// get the kubeconfig from the first master node
	reader, err := runtime.GetKubeconfig(chosenMaster)
	if err != nil {
		log.Errorf("Failed to get kubeconfig from node '%s'", chosenMaster.Name)
		return nil, err
	}
	defer reader.Close()

	readBytes, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Errorln("Couldn't read kubeconfig file")
		return nil, err
	}

	// drop the first 512 bytes which contain file metadata/control characters
	// and trim any NULL characters
	trimBytes := bytes.Trim(readBytes[512:], "\x00")

	/*
	 * Modify the kubeconfig
	 */
	kc, err := kubeconfig.Load(trimBytes)
	if err != nil {
		return nil, err
	}

	// update the server URL
	kc.Clusters["default"].Server = fmt.Sprintf("https://%s:%s", APIHost, APIPort)

	// rename user from default to admin
	newAuthInfoName := fmt.Sprintf("admin@%s-%s", k3d.DefaultObjectNamePrefix, cluster.Name)
	kc.AuthInfos[newAuthInfoName] = kc.AuthInfos["default"]
	delete(kc.AuthInfos, "default")

	// rename cluster from default to clustername
	newClusterName := fmt.Sprintf("%s-%s", k3d.DefaultObjectNamePrefix, cluster.Name)
	kc.Clusters[newClusterName] = kc.Clusters["default"]
	delete(kc.Clusters, "default")

	// rename context from default to admin@clustername
	newContextName := fmt.Sprintf("admin@%s-%s", k3d.DefaultObjectNamePrefix, cluster.Name)
	kc.Contexts[newContextName] = kc.Contexts["default"]
	delete(kc.Contexts, "default")

	// update context with new values for cluster and user
	kc.Contexts[newContextName].AuthInfo = newAuthInfoName
	kc.Contexts[newContextName].Cluster = newClusterName

	// set current-context to new context name
	kc.CurrentContext = newContextName

	log.Debugf("Modified Kubeconfig: %+v", kc)

	trimBytes, err = kubeconfig.Write(*kc)
	if err != nil {
		log.Errorln("Failed to serialize modified kubeconfig")
		return nil, err
	}

	return trimBytes, nil
}

// GetKubeconfigPath uses GetKubeConfig to grab the kubeconfig from the cluster master node, writes it to a file and outputs the path
func GetKubeconfigPath(runtime runtimes.Runtime, cluster *k3d.Cluster, path string) (string, error) {
	var output *os.File
	defer output.Close()
	var err error

	kubeconfigBytes, err := GetKubeconfig(runtime, cluster)
	if err != nil {
		log.Errorln("Failed to get kubeconfig")
		return "", err
	}

	if path == "-" {
		output = os.Stdout
	} else {
		if path == "" {
			basepath, err := util.GetConfigDirOrCreate()
			if err != nil {
				log.Errorln("Failed to create kubeconfig")
				return "", err
			}
			path = fmt.Sprintf("%s/%s-%s.yaml", basepath, k3d.DefaultKubeconfigPrefix, cluster.Name)
		} else if !(strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			log.Warnf("Supplied path '%s' for kubeconfig does not have .yaml or .yml extension", path)
		}
		output, err = os.Create(path)
		if err != nil {
			log.Errorf("Failed to create file '%s'", path)
			return "", err
		}
	}

	_, err = output.Write(kubeconfigBytes)
	if err != nil {
		log.Errorf("Failed to write to file '%s'", output.Name())
		return "", err
	}

	return output.Name(), nil

}
