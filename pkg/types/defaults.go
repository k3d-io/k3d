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
package types

import (
	"fmt"

	"github.com/k3d-io/k3d/v5/pkg/types/k3s"
	"github.com/k3d-io/k3d/v5/version"
)

// DefaultClusterName specifies the default name used for newly created clusters
const DefaultClusterName = "k3s-default"

// DefaultClusterNameMaxLength specifies the maximal length of a passed in cluster name
// This restriction allows us to construct an name consisting of
// <DefaultObjectNamePrefix[3]>-<ClusterName>-<TypeSuffix[5-10]>-<Counter[1-3]>
// ... and still stay within the 64 character limit (e.g. of docker)
const DefaultClusterNameMaxLength = 32

// DefaultObjectNamePrefix defines the name prefix for every object created by k3d
const DefaultObjectNamePrefix = "k3d"

// DefaultRuntimeLabels specifies a set of labels that will be attached to k3d runtime objects by default
var DefaultRuntimeLabels = map[string]string{
	"app": "k3d",
}

// DefaultRuntimeLabelsVar specifies a set of labels that will be attached to k3d runtime objects by default but are not static (e.g. across k3d versions)
var DefaultRuntimeLabelsVar = map[string]string{
	"k3d.version": version.GetVersion(),
}

// DefaultRoleCmds maps the node roles to their respective default commands
var DefaultRoleCmds = map[Role][]string{
	ServerRole: {"server"},
	AgentRole:  {"agent"},
}

// DefaultTmpfsMounts specifies tmpfs mounts that are required for all k3d nodes
var DefaultTmpfsMounts = []string{
	"/run",
	"/var/run",
}

// DefaultNodeEnv defines some default environment variables that should be set on every node
var DefaultNodeEnv = []string{
	fmt.Sprintf("%s=/output/kubeconfig.yaml", k3s.EnvKubeconfigOutput),
}

// DefaultK3dInternalHostRecord defines the default /etc/hosts entry for the k3d host
const DefaultK3dInternalHostRecord = "host.k3d.internal"

// DefaultImageVolumeMountPath defines the mount path inside k3d nodes where we will mount the shared image volume by default
const DefaultImageVolumeMountPath = "/k3d/images"

// DefaultConfigDirName defines the name of the config directory (where we'll e.g. put the kubeconfigs)
const DefaultConfigDirName = ".config/k3d" // should end up in $XDG_CONFIG_HOME

// DefaultKubeconfigPrefix defines the default prefix for kubeconfig files
const DefaultKubeconfigPrefix = DefaultObjectNamePrefix + "-kubeconfig"

// DefaultAPIPort defines the default Kubernetes API Port
const DefaultAPIPort = "6443"

// DefaultAPIHost defines the default host (IP) for the Kubernetes API
const DefaultAPIHost = "0.0.0.0"

// GetDefaultObjectName prefixes the passed name with the default prefix
func GetDefaultObjectName(name string) string {
	return fmt.Sprintf("%s-%s", DefaultObjectNamePrefix, name)
}

// DefaultNodeWaitForLogMessageCrashLoopBackOffLimit defines the maximum number of retries to find the target log message, if the
// container is in a crash loop.
// This makes sense e.g. when a new server is waiting to join an existing cluster and has to wait for other learners to finish.
const DefaultNodeWaitForLogMessageCrashLoopBackOffLimit = 10

// DefaultNetwork defines the default (Docker) runtime network
const DefaultRuntimeNetwork = "bridge"
