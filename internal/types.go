package types

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
