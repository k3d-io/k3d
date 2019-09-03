package types

// DefaultClusterName specifies the default name used for newly created clusters
const DefaultClusterName = "k3s-default"

// DefaultK3sImageRepo specifies the default image repository for the used k3s image
const DefaultK3sImageRepo = "docker.io/rancher/k3s"

// Cluster describes a k3d cluster
type Cluster struct {
	Name    string
	Network string
	Nodes   []Node
}

// Node describes a k3d node
type Node struct {
	Name    string
	Role    string
	Image   string
	Volumes []string
	Env     []string
	Args    []string
	Ports   []string
	Restart bool
}
