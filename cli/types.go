package run

// Node describes a k3d node (= docker container)
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

// Cluster describes a k3d cluster (nodes, combined in a network)
type Cluster struct {
	Name    string
	Network string
	Nodes   []Node
}
