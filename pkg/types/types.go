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
package types

import (
	"context"
	"net"
	"time"

	"github.com/docker/go-connections/nat"
	runtimeTypes "github.com/rancher/k3d/v5/pkg/runtimes/types"
	"github.com/rancher/k3d/v5/pkg/types/k3s"
	"inet.af/netaddr"
)

// NodeStatusRestarting defines the status string that signals the node container is restarting
const NodeStatusRestarting = "restarting"

// Role defines a k3d node role
type Role string

// existing k3d node roles
const (
	ServerRole       Role = "server"
	AgentRole        Role = "agent"
	NoRole           Role = "noRole"
	LoadBalancerRole Role = "loadbalancer"
	RegistryRole     Role = "registry"
)

type InternalRole Role

const (
	InternalRoleInitServer InternalRole = "initServer"
)

// NodeRoles defines the roles available for nodes
var NodeRoles = map[string]Role{
	string(ServerRole):       ServerRole,
	string(AgentRole):        AgentRole,
	string(LoadBalancerRole): LoadBalancerRole,
	string(RegistryRole):     RegistryRole,
}

// ClusterInternalNodeRoles is a list of roles for nodes that belong to a cluster
var ClusterInternalNodeRoles = []Role{
	ServerRole,
	AgentRole,
	LoadBalancerRole,
}

// ClusterExternalNodeRoles is a list of roles for nodes that do not belong to a specific cluster
var ClusterExternalNodeRoles = []Role{
	RegistryRole,
}

// List of k3d technical label name
const (
	LabelClusterName          string = "k3d.cluster"
	LabelClusterURL           string = "k3d.cluster.url"
	LabelClusterToken         string = "k3d.cluster.token"
	LabelClusterExternal      string = "k3d.cluster.external"
	LabelImageVolume          string = "k3d.cluster.imageVolume"
	LabelNetworkExternal      string = "k3d.cluster.network.external"
	LabelNetwork              string = "k3d.cluster.network"
	LabelNetworkID            string = "k3d.cluster.network.id"
	LabelNetworkIPRange       string = "k3d.cluster.network.iprange"
	LabelRole                 string = "k3d.role"
	LabelServerAPIPort        string = "k3d.server.api.port"
	LabelServerAPIHost        string = "k3d.server.api.host"
	LabelServerAPIHostIP      string = "k3d.server.api.hostIP"
	LabelServerIsInit         string = "k3d.server.init"
	LabelRegistryHost         string = "k3d.registry.host"
	LabelRegistryHostIP       string = "k3d.registry.hostIP"
	LabelRegistryPortExternal string = "k3s.registry.port.external"
	LabelRegistryPortInternal string = "k3s.registry.port.internal"
	LabelNodeStaticIP         string = "k3d.node.staticIP"
)

// DoNotCopyServerFlags defines a list of commands/args that shouldn't be copied from an existing node when adding a similar node to a cluster
var DoNotCopyServerFlags = []string{
	"--cluster-init",
}

// ClusterCreateOpts describe a set of options one can set when creating a cluster
type ClusterCreateOpts struct {
	DisableImageVolume  bool              `yaml:"disableImageVolume" json:"disableImageVolume,omitempty"`
	WaitForServer       bool              `yaml:"waitForServer" json:"waitForServer,omitempty"`
	Timeout             time.Duration     `yaml:"timeout" json:"timeout,omitempty"`
	DisableLoadBalancer bool              `yaml:"disableLoadbalancer" json:"disableLoadbalancer,omitempty"`
	GPURequest          string            `yaml:"gpuRequest" json:"gpuRequest,omitempty"`
	ServersMemory       string            `yaml:"serversMemory" json:"serversMemory,omitempty"`
	AgentsMemory        string            `yaml:"agentsMemory" json:"agentsMemory,omitempty"`
	NodeHooks           []NodeHook        `yaml:"nodeHooks,omitempty" json:"nodeHooks,omitempty"`
	GlobalLabels        map[string]string `yaml:"globalLabels,omitempty" json:"globalLabels,omitempty"`
	GlobalEnv           []string          `yaml:"globalEnv,omitempty" json:"globalEnv,omitempty"`
	Registries          struct {
		Create *Registry     `yaml:"create,omitempty" json:"create,omitempty"`
		Use    []*Registry   `yaml:"use,omitempty" json:"use,omitempty"`
		Config *k3s.Registry `yaml:"config,omitempty" json:"config,omitempty"` // registries.yaml (k3s config for containerd registry override)
	} `yaml:"registries,omitempty" json:"registries,omitempty"`
}

// NodeHook is an action that is bound to a specifc stage of a node lifecycle
type NodeHook struct {
	Stage  LifecycleStage `yaml:"stage,omitempty" json:"stage,omitempty"`
	Action NodeHookAction `yaml:"action,omitempty" json:"action,omitempty"`
}

// LifecycleStage defines descriptors for specific stages in the lifecycle of a node or cluster object
type LifecycleStage string

// all defined lifecyclestages
const (
	LifecycleStagePreStart  LifecycleStage = "preStart"
	LifecycleStagePostStart LifecycleStage = "postStart"
)

// ClusterStartOpts describe a set of options one can set when (re-)starting a cluster
type ClusterStartOpts struct {
	WaitForServer   bool
	Timeout         time.Duration
	NodeHooks       []NodeHook `yaml:"nodeHooks,omitempty" json:"nodeHooks,omitempty"`
	EnvironmentInfo *EnvironmentInfo
	Intent          Intent
}

// ClusterDeleteOpts describe a set of options one can set when deleting a cluster
type ClusterDeleteOpts struct {
	SkipRegistryCheck bool // skip checking if this is a registry (and act accordingly)
}

// NodeCreateOpts describes a set of options one can set when creating a new node
type NodeCreateOpts struct {
	Wait            bool
	Timeout         time.Duration
	NodeHooks       []NodeHook `yaml:"nodeHooks,omitempty" json:"nodeHooks,omitempty"`
	EnvironmentInfo *EnvironmentInfo
	ClusterToken    string
}

// NodeStartOpts describes a set of options one can set when (re-)starting a node
type NodeStartOpts struct {
	Wait            bool
	Timeout         time.Duration
	NodeHooks       []NodeHook `yaml:"nodeHooks,omitempty" json:"nodeHooks,omitempty"`
	ReadyLogMessage string
	EnvironmentInfo *EnvironmentInfo
	Intent          Intent
}

// NodeDeleteOpts describes a set of options one can set when deleting a node
type NodeDeleteOpts struct {
	SkipLBUpdate bool // skip updating the loadbalancer
}

// NodeHookAction is an interface to implement actions that should trigger at specific points of the node lifecycle
type NodeHookAction interface {
	Run(ctx context.Context, node *Node) error
	Name() string // returns the name (type) of the action
	Info() string // returns a description of what this action does
}

// LoadMode describes how images are loaded into the cluster
type ImportMode string

const (
	ImportModeAutoDetect ImportMode = "auto"
	ImportModeDirect     ImportMode = "direct"
	ImportModeToolsNode  ImportMode = "tools-node"
)

// ImportModes defines the loading methods for image loading
var ImportModes = map[string]ImportMode{
	string(ImportModeAutoDetect): ImportModeAutoDetect,
	string(ImportModeDirect):     ImportModeDirect,
	string(ImportModeToolsNode):  ImportModeToolsNode,
}

// ImageImportOpts describes a set of options one can set for loading image(s) into cluster(s)
type ImageImportOpts struct {
	KeepTar       bool
	KeepToolsNode bool
	Mode          ImportMode
}

type IPAM struct {
	IPPrefix netaddr.IPPrefix `yaml:"ipPrefix" json:"ipPrefix,omitempty"`
	IPsUsed  []netaddr.IP     `yaml:"ipsUsed" json:"ipsUsed,omitempty"`
	Managed  bool             // IPAM is done by k3d
}

type NetworkMember struct {
	Name string
	IP   netaddr.IP
}

// ClusterNetwork describes a network which a cluster is running in
type ClusterNetwork struct {
	Name     string `yaml:"name" json:"name,omitempty"`
	ID       string `yaml:"id" json:"id"` // may be the same as name, but e.g. docker only differentiates by random ID, not by name
	External bool   `yaml:"external" json:"isExternal,omitempty"`
	IPAM     IPAM   `yaml:"ipam" json:"ipam,omitempty"`
	Members  []*NetworkMember
}

// Cluster describes a k3d cluster
type Cluster struct {
	Name               string             `yaml:"name" json:"name,omitempty"`
	Network            ClusterNetwork     `yaml:"network" json:"network,omitempty"`
	Token              string             `yaml:"clusterToken" json:"clusterToken,omitempty"`
	Nodes              []*Node            `yaml:"nodes" json:"nodes,omitempty"`
	InitNode           *Node              // init server node
	ExternalDatastore  *ExternalDatastore `yaml:"externalDatastore,omitempty" json:"externalDatastore,omitempty"`
	KubeAPI            *ExposureOpts      `yaml:"kubeAPI" json:"kubeAPI,omitempty"`
	ServerLoadBalancer *Loadbalancer      `yaml:"serverLoadbalancer,omitempty" json:"serverLoadBalancer,omitempty"`
	ImageVolume        string             `yaml:"imageVolume" json:"imageVolume,omitempty"`
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

type NodeIP struct {
	IP     netaddr.IP
	Static bool
}

// Node describes a k3d node
type Node struct {
	Name          string            `yaml:"name" json:"name,omitempty"`
	Role          Role              `yaml:"role" json:"role,omitempty"`
	Image         string            `yaml:"image" json:"image,omitempty"`
	Volumes       []string          `yaml:"volumes" json:"volumes,omitempty"`
	Env           []string          `yaml:"env" json:"env,omitempty"`
	Cmd           []string          // filled automatically based on role
	Args          []string          `yaml:"extraArgs" json:"extraArgs,omitempty"`
	Ports         nat.PortMap       `yaml:"portMappings" json:"portMappings,omitempty"`
	Restart       bool              `yaml:"restart" json:"restart,omitempty"`
	Created       string            `yaml:"created" json:"created,omitempty"`
	RuntimeLabels map[string]string `yaml:"runtimeLabels" json:"runtimeLabels,omitempty"`
	K3sNodeLabels map[string]string `yaml:"k3sNodeLabels" json:"k3sNodeLabels,omitempty"`
	Networks      []string          // filled automatically
	ExtraHosts    []string          // filled automatically
	ServerOpts    ServerOpts        `yaml:"serverOpts" json:"serverOpts,omitempty"`
	AgentOpts     AgentOpts         `yaml:"agentOpts" json:"agentOpts,omitempty"`
	GPURequest    string            // filled automatically
	Memory        string            // filled automatically
	State         NodeState         // filled automatically
	IP            NodeIP            // filled automatically -> refers solely to the cluster network
	HookActions   []NodeHook        `yaml:"hooks" json:"hooks,omitempty"`
}

// ServerOpts describes some additional server role specific opts
type ServerOpts struct {
	IsInit  bool          `yaml:"isInitializingServer" json:"isInitializingServer,omitempty"`
	KubeAPI *ExposureOpts `yaml:"kubeAPI" json:"kubeAPI"`
}

// ExposureOpts describes settings that the user can set for accessing the Kubernetes API
type ExposureOpts struct {
	nat.PortMapping        // filled automatically (reference to normal portmapping)
	Host            string `yaml:"host,omitempty" json:"host,omitempty"`
}

// ExternalDatastore describes an external datastore used for HA/multi-server clusters
type ExternalDatastore struct {
	Endpoint string `yaml:"endpoint" json:"endpoint,omitempty"`
	CAFile   string `yaml:"caFile" json:"caFile,omitempty"`
	CertFile string `yaml:"certFile" json:"certFile,omitempty"`
	KeyFile  string `yaml:"keyFile" json:"keyFile,omitempty"`
	Network  string `yaml:"network" json:"network,omitempty"`
}

// AgentOpts describes some additional agent role specific opts
type AgentOpts struct{}

// NodeState describes the current state of a node
type NodeState struct {
	Running bool
	Status  string
	Started string
}

type EnvironmentInfo struct {
	HostGateway net.IP
	RuntimeInfo runtimeTypes.RuntimeInfo
}
