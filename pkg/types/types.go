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
	"context"
	"net/netip"
	"time"

	"github.com/docker/go-connections/nat"

	dockerunits "github.com/docker/go-units"
	runtimeTypes "github.com/k3d-io/k3d/v5/pkg/runtimes/types"
	wharfie "github.com/rancher/wharfie/pkg/registries"
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
	LabelClusterName             string = "k3d.cluster"
	LabelClusterURL              string = "k3d.cluster.url"
	LabelClusterToken            string = "k3d.cluster.token"
	LabelClusterExternal         string = "k3d.cluster.external"
	LabelImageVolume             string = "k3d.cluster.imageVolume"
	LabelNetworkExternal         string = "k3d.cluster.network.external"
	LabelNetwork                 string = "k3d.cluster.network"
	LabelNetworkID               string = "k3d.cluster.network.id"
	LabelNetworkIPRange          string = "k3d.cluster.network.iprange"
	LabelClusterStartHostAliases string = "k3d.cluster.start.hostaliases"
	LabelRole                    string = "k3d.role"
	LabelServerAPIPort           string = "k3d.server.api.port"
	LabelServerAPIHost           string = "k3d.server.api.host"
	LabelServerAPIHostIP         string = "k3d.server.api.hostIP"
	LabelServerIsInit            string = "k3d.server.init"
	LabelServerLoadBalancer      string = "k3d.server.loadbalancer"
	LabelRegistryHost            string = "k3d.registry.host"
	LabelRegistryHostIP          string = "k3d.registry.hostIP"
	LabelRegistryPortExternal    string = "k3s.registry.port.external"
	LabelRegistryPortInternal    string = "k3s.registry.port.internal"
	LabelNodeStaticIP            string = "k3d.node.staticIP"
)

// DoNotCopyServerFlags defines a list of commands/args that shouldn't be copied from an existing node when adding a similar node to a cluster
var DoNotCopyServerFlags = []string{
	"--cluster-init",
}

type HostAlias struct {
	IP        string   `mapstructure:"ip" json:"ip"`
	Hostnames []string `mapstructure:"hostnames" json:"hostnames"`
}

// ClusterCreateOpts describe a set of options one can set when creating a cluster
type ClusterCreateOpts struct {
	DisableImageVolume  bool              `json:"disableImageVolume,omitempty"`
	WaitForServer       bool              `json:"waitForServer,omitempty"`
	Timeout             time.Duration     `json:"timeout,omitempty"`
	DisableLoadBalancer bool              `json:"disableLoadbalancer,omitempty"`
	GPURequest          string            `json:"gpuRequest,omitempty"`
	ServersMemory       string            `json:"serversMemory,omitempty"`
	AgentsMemory        string            `json:"agentsMemory,omitempty"`
	NodeHooks           []NodeHook        `json:"nodeHooks,omitempty"`
	GlobalLabels        map[string]string `json:"globalLabels,omitempty"`
	GlobalEnv           []string          `json:"globalEnv,omitempty"`
	HostAliases         []HostAlias       `json:"hostAliases,omitempty"`
	Registries          struct {
		Create *Registry         `json:"create,omitempty"`
		Use    []*Registry       `json:"use,omitempty"`
		Config *wharfie.Registry `json:"config,omitempty"` // registries.yaml (k3s config for containerd registry override)
	} `json:"registries,omitempty"`
}

// NodeHook is an action that is bound to a specifc stage of a node lifecycle
type NodeHook struct {
	Stage  LifecycleStage `json:"stage,omitempty"`
	Action NodeHookAction `json:"action,omitempty"`
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
	NodeHooks       []NodeHook `json:"nodeHooks,omitempty"`
	EnvironmentInfo *EnvironmentInfo
	Intent          Intent
	HostAliases     []HostAlias `json:"hostAliases,omitempty"`
}

// ClusterDeleteOpts describe a set of options one can set when deleting a cluster
type ClusterDeleteOpts struct {
	SkipRegistryCheck bool // skip checking if this is a registry (and act accordingly)
}

// NodeCreateOpts describes a set of options one can set when creating a new node
type NodeCreateOpts struct {
	Wait            bool
	Timeout         time.Duration
	NodeHooks       []NodeHook `json:"nodeHooks,omitempty"`
	EnvironmentInfo *EnvironmentInfo
	ClusterToken    string
}

// NodeStartOpts describes a set of options one can set when (re-)starting a node
type NodeStartOpts struct {
	Wait            bool
	Timeout         time.Duration
	NodeHooks       []NodeHook `json:"nodeHooks,omitempty"`
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
	IPPrefix netip.Prefix `json:"ipPrefix,omitempty"`
	IPsUsed  []netip.Addr `json:"ipsUsed,omitempty"`
	Managed  bool         // IPAM is done by k3d
}

type NetworkMember struct {
	Name string
	IP   netip.Addr
}

// ClusterNetwork describes a network which a cluster is running in
type ClusterNetwork struct {
	Name     string `json:"name,omitempty"`
	ID       string `json:"id"` // may be the same as name, but e.g. docker only differentiates by random ID, not by name
	External bool   `json:"isExternal,omitempty"`
	IPAM     IPAM   `json:"ipam,omitempty"`
	Members  []*NetworkMember
}

// Cluster describes a k3d cluster
type Cluster struct {
	Name               string             `json:"name,omitempty"`
	Network            ClusterNetwork     `json:"network,omitempty"`
	Token              string             `json:"clusterToken,omitempty"`
	Nodes              []*Node            `json:"nodes,omitempty"`
	InitNode           *Node              // init server node
	ExternalDatastore  *ExternalDatastore `json:"externalDatastore,omitempty"`
	KubeAPI            *ExposureOpts      `json:"kubeAPI,omitempty"`
	ServerLoadBalancer *Loadbalancer      `json:"serverLoadBalancer,omitempty"`
	ImageVolume        string             `json:"imageVolume,omitempty"`
	Volumes            []string           `json:"volumes,omitempty"` // k3d-managed volumes attached to this cluster
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
	IP     netip.Addr
	Static bool
}

// Node describes a k3d node
type Node struct {
	Name           string                `json:"name,omitempty"`
	Role           Role                  `json:"role,omitempty"`
	Image          string                `json:"image,omitempty"`
	Volumes        []string              `json:"volumes,omitempty"`
	Env            []string              `json:"env,omitempty"`
	Cmd            []string              // filled automatically based on role
	Args           []string              `json:"extraArgs,omitempty"`
	Files          []File                `json:"files,omitempty"`
	Ports          nat.PortMap           `json:"portMappings,omitempty"`
	Restart        bool                  `json:"restart,omitempty"`
	Created        string                `json:"created,omitempty"`
	HostPidMode    bool                  `json:"hostPidMode,omitempty"`
	RuntimeLabels  map[string]string     `json:"runtimeLabels,omitempty"`
	RuntimeUlimits []*dockerunits.Ulimit `json:"runtimeUlimits,omitempty"`
	K3sNodeLabels  map[string]string     `json:"k3sNodeLabels,omitempty"`
	Networks       []string              // filled automatically
	ExtraHosts     []string              // filled automatically (docker specific?)
	ServerOpts     ServerOpts            `json:"serverOpts,omitempty"`
	AgentOpts      AgentOpts             `json:"agentOpts,omitempty"`
	GPURequest     string                // filled automatically
	Memory         string                // filled automatically
	State          NodeState             // filled automatically
	IP             NodeIP                // filled automatically -> refers solely to the cluster network
	HookActions    []NodeHook            `json:"hooks,omitempty"`
	K3dEntrypoint  bool
}

// ServerOpts describes some additional server role specific opts
type ServerOpts struct {
	IsInit  bool          `json:"isInitializingServer,omitempty"`
	KubeAPI *ExposureOpts `json:"kubeAPI"`
}

// ExposureOpts describes settings that the user can set for accessing the Kubernetes API
type ExposureOpts struct {
	nat.PortMapping        // filled automatically (reference to normal portmapping)
	Host            string `json:"host,omitempty"`
}

// ExternalDatastore describes an external datastore used for HA/multi-server clusters
type ExternalDatastore struct {
	Endpoint string `json:"endpoint,omitempty"`
	CAFile   string `json:"caFile,omitempty"`
	CertFile string `json:"certFile,omitempty"`
	KeyFile  string `json:"keyFile,omitempty"`
	Network  string `json:"network,omitempty"`
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
	HostGateway netip.Addr
	RuntimeInfo runtimeTypes.RuntimeInfo
}
