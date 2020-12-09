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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/rancher/k3d/v4/pkg/runtimes"
	k3d "github.com/rancher/k3d/v4/pkg/types"
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

// KubeconfigGetWrite ...
// 1. fetches the KubeConfig from the first server node retrieved for a given cluster
// 2. modifies it by updating some fields with cluster-specific information
// 3. writes it to the specified output
func KubeconfigGetWrite(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster, output string, writeKubeConfigOptions *WriteKubeConfigOptions) (string, error) {

	// get kubeconfig from cluster node
	kubeconfig, err := KubeconfigGet(ctx, runtime, cluster)
	if err != nil {
		return output, err
	}

	// empty output parameter = write to default
	if output == "" {
		output, err = KubeconfigGetDefaultPath()
		if err != nil {
			return output, err
		}
	}

	// simply write to the output, ignoring existing contents
	if writeKubeConfigOptions.OverwriteExisting || output == "-" {
		return output, KubeconfigWriteToPath(ctx, kubeconfig, output)
	}

	// load config from existing file or fail if it has non-kubeconfig contents
	var existingKubeConfig *clientcmdapi.Config
	firstRun := true
	for {
		existingKubeConfig, err = clientcmd.LoadFromFile(output) // will return an empty config if file is empty
		if err != nil {

			// the output file does not exist: try to create it and try again
			if os.IsNotExist(err) && firstRun {
				log.Debugf("Output path '%s' doesn't exist, trying to create it...", output)

				// create directory path
				if err := os.MkdirAll(path.Dir(output), 0755); err != nil {
					log.Errorf("Failed to create output directory '%s'", path.Dir(output))
					return output, err
				}

				// create output file
				if _, err := os.Create(output); err != nil {
					log.Errorf("Failed to create output file '%s'", output)
					return output, err
				}

				// try again, but do not try to create the file this time
				firstRun = false
				continue
			}
			log.Errorf("Failed to open output file '%s' or it's not a KubeConfig", output)
			return output, err
		}
		break
	}

	// update existing kubeconfig, but error out if there are conflicting fields but we don't want to update them
	return output, KubeconfigMerge(ctx, kubeconfig, existingKubeConfig, output, writeKubeConfigOptions.UpdateExisting, writeKubeConfigOptions.UpdateCurrentContext)

}

// KubeconfigGet grabs the kubeconfig file from /output from a server node container,
// modifies it by updating some fields with cluster-specific information
// and returns a Config object for further processing
func KubeconfigGet(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster) (*clientcmdapi.Config, error) {
	// get all server nodes for the selected cluster
	// TODO: getKubeconfig: we should make sure, that the server node we're trying to fetch from is actually running
	serverNodes, err := runtime.GetNodesByLabel(ctx, map[string]string{k3d.LabelClusterName: cluster.Name, k3d.LabelRole: string(k3d.ServerRole)})
	if err != nil {
		log.Errorln("Failed to get server nodes")
		return nil, err
	}
	if len(serverNodes) == 0 {
		return nil, fmt.Errorf("Didn't find any server node")
	}

	// prefer a server node, which actually has the port exposed
	var chosenServer *k3d.Node
	chosenServer = nil
	APIPort := k3d.DefaultAPIPort
	APIHost := k3d.DefaultAPIHost

	for _, server := range serverNodes {
		if _, ok := server.Labels[k3d.LabelServerAPIPort]; ok {
			chosenServer = server
			APIPort = server.Labels[k3d.LabelServerAPIPort]
			if _, ok := server.Labels[k3d.LabelServerAPIHost]; ok {
				APIHost = server.Labels[k3d.LabelServerAPIHost]
			}
			break
		}
	}

	if chosenServer == nil {
		chosenServer = serverNodes[0]
	}
	// get the kubeconfig from the first server node
	reader, err := runtime.GetKubeconfig(ctx, chosenServer)
	if err != nil {
		log.Errorf("Failed to get kubeconfig from node '%s'", chosenServer.Name)
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

	// rename context from default to clustername
	newContextName := fmt.Sprintf("%s-%s", k3d.DefaultObjectNamePrefix, cluster.Name)
	kc.Contexts[newContextName] = kc.Contexts["default"]
	delete(kc.Contexts, "default")

	// update context with new values for cluster and user
	kc.Contexts[newContextName].AuthInfo = newAuthInfoName
	kc.Contexts[newContextName].Cluster = newClusterName

	// set current-context to new context name
	kc.CurrentContext = newContextName

	log.Tracef("Modified Kubeconfig: %+v", kc)

	return kc, nil
}

// KubeconfigWriteToPath takes a kubeconfig and writes it to some path, which can be '-' for os.Stdout
func KubeconfigWriteToPath(ctx context.Context, kubeconfig *clientcmdapi.Config, path string) error {
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

	log.Debugf("Wrote kubeconfig to '%s'", output.Name())

	return nil

}

// KubeconfigMerge merges a new kubeconfig into an existing kubeconfig and returns the result
func KubeconfigMerge(ctx context.Context, newKubeConfig *clientcmdapi.Config, existingKubeConfig *clientcmdapi.Config, outPath string, overwriteConflicting bool, updateCurrentContext bool) error {

	log.Tracef("Merging new Kubeconfig:\n%+v\n>>> into existing Kubeconfig:\n%+v", newKubeConfig, existingKubeConfig)

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
		if _, ok := existingKubeConfig.Contexts[k]; ok && !overwriteConflicting {
			return fmt.Errorf("Context '%s' already exists in target KubeConfig", k)
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

	kubeconfigYaml, err := clientcmd.Write(*existingKubeConfig)
	if err != nil {
		log.Debugf("Merged Kubeconfig:\n%+v", existingKubeConfig)
	} else {
		log.Tracef("Merged Kubeconfig:\n%s", kubeconfigYaml)
	}

	return KubeconfigWrite(ctx, existingKubeConfig, outPath)
}

// KubeconfigWrite writes a kubeconfig to a path atomically
func KubeconfigWrite(ctx context.Context, kubeconfig *clientcmdapi.Config, path string) error {
	tempPath := fmt.Sprintf("%s.k3d_%s", path, time.Now().Format("20060102_150405.000000"))
	if err := clientcmd.WriteToFile(*kubeconfig, tempPath); err != nil {
		log.Errorf("Failed to write merged kubeconfig to temporary file '%s'", tempPath)
		return err
	}

	// Move temporary file over existing KubeConfig
	if err := os.Rename(tempPath, path); err != nil {
		log.Errorf("Failed to overwrite existing KubeConfig '%s' with new KubeConfig '%s'", path, tempPath)
		return err
	}

	log.Debugf("Wrote kubeconfig to '%s'", path)

	return nil
}

// KubeconfigGetDefaultFile loads the default KubeConfig file
func KubeconfigGetDefaultFile() (*clientcmdapi.Config, error) {
	path, err := KubeconfigGetDefaultPath()
	if err != nil {
		return nil, err
	}
	log.Debugf("Using default kubeconfig '%s'", path)
	return clientcmd.LoadFromFile(path)
}

// KubeconfigGetDefaultPath returns the path of the default kubeconfig, but errors if the KUBECONFIG env var specifies more than one file
func KubeconfigGetDefaultPath() (string, error) {
	defaultKubeConfigLoadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if len(defaultKubeConfigLoadingRules.GetLoadingPrecedence()) > 1 {
		return "", fmt.Errorf("Multiple kubeconfigs specified via KUBECONFIG env var: Please reduce to one entry, unset KUBECONFIG or explicitly choose an output")
	}
	return defaultKubeConfigLoadingRules.GetDefaultFilename(), nil
}

// KubeconfigRemoveClusterFromDefaultConfig removes a cluster's details from the default kubeconfig
func KubeconfigRemoveClusterFromDefaultConfig(ctx context.Context, cluster *k3d.Cluster) error {
	defaultKubeConfigPath, err := KubeconfigGetDefaultPath()
	if err != nil {
		return err
	}
	kubeconfig, err := KubeconfigGetDefaultFile()
	if err != nil {
		return err
	}
	kubeconfig = KubeconfigRemoveCluster(ctx, cluster, kubeconfig)
	return KubeconfigWrite(ctx, kubeconfig, defaultKubeConfigPath)
}

// KubeconfigRemoveCluster removes a cluster's details from a given kubeconfig
func KubeconfigRemoveCluster(ctx context.Context, cluster *k3d.Cluster, kubeconfig *clientcmdapi.Config) *clientcmdapi.Config {
	clusterName := fmt.Sprintf("%s-%s", k3d.DefaultObjectNamePrefix, cluster.Name)
	contextName := fmt.Sprintf("%s-%s", k3d.DefaultObjectNamePrefix, cluster.Name)
	authInfoName := fmt.Sprintf("admin@%s-%s", k3d.DefaultObjectNamePrefix, cluster.Name)

	// delete elements from kubeconfig if they're present
	delete(kubeconfig.Contexts, contextName)
	delete(kubeconfig.Clusters, clusterName)
	delete(kubeconfig.AuthInfos, authInfoName)

	// set current-context to any other context, if it was set to the given cluster before
	if kubeconfig.CurrentContext == contextName {
		for k := range kubeconfig.Contexts {
			kubeconfig.CurrentContext = k
			break
		}
		// if current-context didn't change, unset it
		if kubeconfig.CurrentContext == contextName {
			kubeconfig.CurrentContext = ""
		}
	}
	return kubeconfig
}
