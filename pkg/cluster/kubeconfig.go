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
	"time"

	"github.com/rancher/k3d/pkg/runtimes"
	k3d "github.com/rancher/k3d/pkg/types"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// WriteKubeConfigOptions provide a set of options for writing a KubeConfig file
type WriteKubeConfigOptions struct {
	UpdateExisting       bool
	UpdateCurrentContext bool
	OverwriteExisting    bool
}

// GetAndWriteKubeConfig ...
// 1. fetches the KubeConfig from the first master node retrieved for a given cluster
// 2. modifies it by updating some fields with cluster-specific information
// 3. writes it to the specified output
func GetAndWriteKubeConfig(runtime runtimes.Runtime, cluster *k3d.Cluster, output string, writeKubeConfigOptions *WriteKubeConfigOptions) error {

	// get kubeconfig from cluster node
	kubeconfig, err := GetKubeconfig(runtime, cluster)
	if err != nil {
		return err
	}

	// empty output parameter = write to default
	if output == "" {
		defaultKubeConfigLoadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		if len(defaultKubeConfigLoadingRules.GetLoadingPrecedence()) > 1 {
			return fmt.Errorf("Multiple kubeconfigs specified via KUBECONFIG env var: Please reduce to one entry, unset KUBECONFIG or explicitly choose an output")
		}
		output = defaultKubeConfigLoadingRules.GetDefaultFilename()
	}

	// simply write to the output, ignoring existing contents
	if writeKubeConfigOptions.OverwriteExisting || output == "-" {
		return WriteKubeConfigToPath(kubeconfig, output)
	}

	// load config from existing file or fail if it has non-kubeconfig contents
	var existingKubeConfig *clientcmdapi.Config
	firstRun := true
	for {
		existingKubeConfig, err = clientcmd.LoadFromFile(output) // will return an empty config if file is empty
		if err != nil {
			if os.IsNotExist(err) && firstRun {
				if _, err := os.Create(output); err != nil {
					log.Errorln("Failed to create output file")
					return err
				}
				firstRun = false
				continue
			}
			log.Errorf("Failed to open output file '%s' or it's not a KubeConfig", output)
			return err
		}
		break
	}

	// update existing kubeconfig, but error out if there are conflicting fields but we don't want to update them
	return UpdateKubeConfig(kubeconfig, existingKubeConfig, output, writeKubeConfigOptions.UpdateExisting, writeKubeConfigOptions.UpdateCurrentContext)

}

// GetKubeconfig grabs the kubeconfig file from /output from a master node container,
// modifies it by updating some fields with cluster-specific information
// and returns a Config object for further processing
func GetKubeconfig(runtime runtimes.Runtime, cluster *k3d.Cluster) (*clientcmdapi.Config, error) {
	// get all master nodes for the selected cluster
	// TODO: getKubeconfig: we should make sure, that the master node we're trying to fetch from is actually running
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
	kc, err := clientcmd.Load(trimBytes)
	if err != nil {
		log.Errorln("Failed to parse the KubeConfig")
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

	return kc, nil
}

// WriteKubeConfigToPath takes a kubeconfig and writes it to some path, which can be '-' for os.Stdout
func WriteKubeConfigToPath(kubeconfig *clientcmdapi.Config, path string) error {
	var output *os.File
	defer output.Close()
	var err error

	if path == "-" {
		output = os.Stdout
	} else {
		output, err = os.Create(path)
		if err != nil {
			log.Errorf("Failed to create file '%s'", path)
			return err
		}
		defer output.Close()
	}

	kubeconfigBytes, err := clientcmd.Write(*kubeconfig)
	if err != nil {
		log.Errorln("Failed to write KubeConfig")
		return err
	}

	_, err = output.Write(kubeconfigBytes)
	if err != nil {
		log.Errorf("Failed to write to file '%s'", output.Name())
		return err
	}

	return nil

}

// UpdateKubeConfig merges a new kubeconfig into an existing kubeconfig and returns the result
func UpdateKubeConfig(newKubeConfig *clientcmdapi.Config, existingKubeConfig *clientcmdapi.Config, outPath string, overwriteConflicting bool, updateCurrentContext bool) error {

	log.Debugf("Merging new KubeConfig:\n%+v\n>>> into existing KubeConfig:\n%+v", newKubeConfig, existingKubeConfig)

	// Overwrite values in existing kubeconfig
	for k, v := range newKubeConfig.Clusters {
		if _, ok := existingKubeConfig.Clusters[k]; ok {
			if !overwriteConflicting {
				return fmt.Errorf("Cluster '%s' already exists in target KubeConfig", k)
			}
		}
		existingKubeConfig.Clusters[k] = v
	}

	for k, v := range newKubeConfig.AuthInfos {
		if _, ok := existingKubeConfig.AuthInfos[k]; ok {
			if !overwriteConflicting {
				return fmt.Errorf("AuthInfo '%s' already exists in target KubeConfig", k)
			}
		}
		existingKubeConfig.AuthInfos[k] = v
	}

	for k, v := range newKubeConfig.Contexts {
		if _, ok := existingKubeConfig.Clusters[k]; ok && !overwriteConflicting {
			return fmt.Errorf("Cluster '%s' already exists in target KubeConfig", k)
		}
		existingKubeConfig.Contexts[k] = v
	}

	// Set current context if it's
	// a) empty
	// b) not empty, but we want to update it
	if existingKubeConfig.CurrentContext == "" {
		updateCurrentContext = true
	}
	if updateCurrentContext {
		log.Debugf("Setting new current-context '%s'", newKubeConfig.CurrentContext)
		existingKubeConfig.CurrentContext = newKubeConfig.CurrentContext
	}

	log.Debugf("Merged KubeConfig:\n%+v", existingKubeConfig)

	// Write updated/merged KubeConfig to new temporary file
	mergedConfigPath := fmt.Sprintf("%s.k3d_%s", outPath, time.Now().Format("20060102_150405.000000"))
	if err := clientcmd.WriteToFile(*existingKubeConfig, mergedConfigPath); err != nil {
		log.Errorf("Failed to write merged kubeconfig to temporary file '%s'", mergedConfigPath)
		return err
	}

	// Move temporary file over existing KubeConfig
	if err := os.Rename(mergedConfigPath, outPath); err != nil {
		log.Errorf("Failed to overwrite existing KubeConfig '%s' with new KubeConfig '%s'", outPath, mergedConfigPath)
		return err
	}

	return nil
}
