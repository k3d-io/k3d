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
package types

import (
	"fmt"
	"time"
)

// DefaultClusterName specifies the default name used for newly created clusters
const DefaultClusterName = "k3s-default"

// DefaultClusterNameMaxLength specifies the maximal length of a passed in cluster name
// This restriction allows us to construct an name consisting of
// <DefaultObjectNamePrefix[3]>-<ClusterName>-<TypeSuffix[5-10]>-<Counter[1-3]>
// ... and still stay within the 64 character limit (e.g. of docker)
const DefaultClusterNameMaxLength = 32

// DefaultK3sImageRepo specifies the default image repository for the used k3s image
const DefaultK3sImageRepo = "docker.io/rancher/k3s"

// DefaultLBImageRepo defines the default cluster load balancer image
const DefaultLBImageRepo = "docker.io/rancher/k3d-proxy"

// DefaultToolsImageRepo defines the default image used for the tools container
const DefaultToolsImageRepo = "docker.io/rancher/k3d-tools"

// DefaultObjectNamePrefix defines the name prefix for every object created by k3d
const DefaultObjectNamePrefix = "k3d"

// ReadyLogMessageByRole defines the log messages we wait for until a server node is considered ready
var ReadyLogMessageByRole = map[Role]string{
	ServerRole:       "Wrote kubeconfig",
	AgentRole:        "Successfully registered node",
	LoadBalancerRole: "start worker processes",
}

// Role defines a k3d node role
type Role string

// existing k3d node roles
const (
	ServerRole       Role = "server"
	AgentRole        Role = "agent"
	NoRole           Role = "noRole"
	LoadBalancerRole Role = "loadbalancer"
)

// NodeRoles defines the roles available for nodes
var NodeRoles = map[string]Role{
	string(ServerRole):       ServerRole,
	string(AgentRole):        AgentRole,
	string(LoadBalancerRole): LoadBalancerRole,
}

// DefaultObjectLabels specifies a set of labels that will be attached to k3d objects by default
var DefaultObjectLabels = map[string]string{
	"app": "k3d",
}

// List of k3d technical label name
const (
	LabelClusterName     string = "k3d.cluster"
	LabelClusterURL      string = "k3d.cluster.url"
	LabelClusterToken    string = "k3d.cluster.token"
	LabelImageVolume     string = "k3d.cluster.imageVolume"
	LabelNetworkExternal string = "k3d.cluster.network.external"
	LabelNetwork         string = "k3d.cluster.network"
	LabelRole            string = "k3d.role"
	LabelServerAPIPort   string = "k3d.server.api.port"
	LabelServerAPIHost   string = "k3d.server.api.host"
	LabelServerAPIHostIP string = "k3d.server.api.hostIP"
)

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
	"K3S_KUBECONFIG_OUTPUT=/output/kubeconfig.yaml",
}

// DefaultImageVolumeMountPath defines the mount path inside k3d nodes where we will mount the shared image volume by default
const DefaultImageVolumeMountPath = "/k3d/images"

// DefaultConfigDirName defines the name of the config directory (where we'll e.g. put the kubeconfigs)
const DefaultConfigDirName = ".k3d" // should end up in $HOME/

// DefaultKubeconfigPrefix defines the default prefix for kubeconfig files
const DefaultKubeconfigPrefix = DefaultObjectNamePrefix + "-kubeconfig"

// DefaultAPIPort defines the default Kubernetes API Port
const DefaultAPIPort = "6443"

// DefaultAPIHost defines the default host (IP) for the Kubernetes API
const DefaultAPIHost = "0.0.0.0"

// DoNotCopyServerFlags defines a list of commands/args that shouldn't be copied from an existing node when adding a similar node to a cluster
var DoNotCopyServerFlags = []string{
	"--cluster-init",
}

// ClusterCreateOpts describe a set of options one can set when creating a cluster
type ClusterCreateOpts struct {
	DisableImageVolume  bool
	WaitForServer       bool
	Timeout             time.Duration
	DisableLoadBalancer bool
	K3sServerArgs       []string
	K3sAgentArgs        []string
}

// ClusterStartOpts describe a set of options one can set when (re-)starting a cluster
type ClusterStartOpts struct {
	WaitForServer bool
	Timeout       time.Duration
}

// NodeCreateOpts describes a set of options one can set when creating a new node
type NodeCreateOpts struct {
	Wait    bool
	Timeout time.Duration
}

// NodeStartOpts describes a set of options one can set when (re-)starting a node
type NodeStartOpts struct {
	Wait    bool
	Timeout time.Duration
}

// ImageImportOpts describes a set of options one can set for loading image(s) into cluster(s)
type ImageImportOpts struct {
	KeepTar bool
}

// ClusterNetwork describes a network which a cluster is running in
type ClusterNetwork struct {
	Name     string `yaml:"name" json:"name,omitempty"`
	External bool   `yaml:"external" json:"isExternal,omitempty"`
}

// Cluster describes a k3d cluster
type Cluster struct {
	Name               string             `yaml:"name" json:"name,omitempty"`
	Network            ClusterNetwork     `yaml:"network" json:"network,omitempty"`
	Token              string             `yaml:"cluster_token" json:"clusterToken,omitempty"`
	Nodes              []*Node            `yaml:"nodes" json:"nodes,omitempty"`
	InitNode           *Node              // init server node
	ExternalDatastore  ExternalDatastore  `yaml:"external_datastore" json:"externalDatastore,omitempty"`
	CreateClusterOpts  *ClusterCreateOpts `yaml:"options" json:"options,omitempty"`
	ExposeAPI          ExposeAPI          `yaml:"expose_api" json:"exposeAPI,omitempty"`
	ServerLoadBalancer *Node              `yaml:"server_loadbalancer" json:"serverLoadBalancer,omitempty"`
	ImageVolume        string             `yaml:"image_volume" json:"imageVolume,omitempty"`
}

// ServerCountRunning returns the number of server nodes running in the cluster and the total number
func (c *Cluster) ServerCountRunning() (int, int) {
	serverCount := 0
	serversRunning := 0
	for _, node := range c.Nodes {
		if node.Role == ServerRole {
			serverCount++
			if node.State.Running {
				serversRunning++
			}
		}
	}
	return serverCount, serversRunning
}

// AgentCountRunning returns the number of agent nodes running in the cluster and the total number
func (c *Cluster) AgentCountRunning() (int, int) {
	agentCount := 0
	agentsRunning := 0
	for _, node := range c.Nodes {
		if node.Role == AgentRole {
			agentCount++
			if node.State.Running {
				agentsRunning++
			}
		}
	}
	return agentCount, agentsRunning
}

// HasLoadBalancer returns true if cluster has a loadbalancer node
func (c *Cluster) HasLoadBalancer() bool {
	for _, node := range c.Nodes {
		if node.Role == LoadBalancerRole {
			return true
		}
	}
	return false
}

// Node describes a k3d node
type Node struct {
	Name       string            `yaml:"name" json:"name,omitempty"`
	Role       Role              `yaml:"role" json:"role,omitempty"`
	Image      string            `yaml:"image" json:"image,omitempty"`
	Volumes    []string          `yaml:"volumes" json:"volumes,omitempty"`
	Env        []string          `yaml:"env" json:"env,omitempty"`
	Cmd        []string          // filled automatically based on role
	Args       []string          `yaml:"extra_args" json:"extraArgs,omitempty"`
	Ports      []string          `yaml:"port_mappings" json:"portMappings,omitempty"`
	Restart    bool              `yaml:"restart" json:"restart,omitempty"`
	Labels     map[string]string // filled automatically
	Network    string            // filled automatically
	ServerOpts ServerOpts        `yaml:"server_opts" json:"serverOpts,omitempty"`
	AgentOpts  AgentOpts         `yaml:"agent_opts" json:"agentOpts,omitempty"`
	State      NodeState         // filled automatically
}

// ServerOpts describes some additional server role specific opts
type ServerOpts struct {
	IsInit    bool      `yaml:"is_initializing_server" json:"isInitializingServer,omitempty"`
	ExposeAPI ExposeAPI // filled automatically
}

// ExternalDatastore describes an external datastore used for HA/multi-server clusters
type ExternalDatastore struct {
	Endpoint string `yaml:"endpoint" json:"endpoint,omitempty"`
	CAFile   string `yaml:"ca_file" json:"caFile,omitempty"`
	CertFile string `yaml:"cert_file" json:"certFile,omitempty"`
	KeyFile  string `yaml:"key_file" json:"keyFile,omitempty"`
	Network  string `yaml:"network" json:"network,omitempty"`
}

// ExposeAPI describes specs needed to expose the API-Server
type ExposeAPI struct {
	Host   string `yaml:"host" json:"host,omitempty"`
	HostIP string `yaml:"host_ip" json:"hostIP,omitempty"`
	Port   string `yaml:"port" json:"port"`
}

// AgentOpts describes some additional agent role specific opts
type AgentOpts struct{}

// GetDefaultObjectName prefixes the passed name with the default prefix
func GetDefaultObjectName(name string) string {
	return fmt.Sprintf("%s-%s", DefaultObjectNamePrefix, name)
}

// NodeState describes the current state of a node
type NodeState struct {
	Running bool
	Status  string
}
