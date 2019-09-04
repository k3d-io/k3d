package run

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/go-connections/nat"
)

// Cluster describes an existing cluster
type Cluster struct {
	name        string
	image       string
	status      string
	serverPorts []string
	server      types.Container
	workers     []types.Container
}

// ClusterSpec defines the specs for a cluster that's up for creation
type ClusterSpec struct {
	AgentArgs         []string
	APIPort           apiPort
	AutoRestart       bool
	ClusterName       string
	Env               []string
	Image             string
	NodeToPortSpecMap map[string][]string
	PortAutoOffset    int
	ServerArgs        []string
	Verbose           bool
	Volumes           []string
}

// PublishedPorts is a struct used for exposing container ports on the host system
type PublishedPorts struct {
	ExposedPorts map[nat.Port]struct{}
	PortBindings map[nat.Port][]nat.PortBinding
}
