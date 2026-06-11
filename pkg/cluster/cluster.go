package cluster

import (
    "github.com/k3d-io/k3d/v5/pkg/types"
)

type Cluster struct {
    // ...
    IP string // Add ip field
}

func NewCluster(name string, servers, agents []string, port, apiPort int, ip string) *Cluster {
    return &Cluster{
        // ...
        IP: ip, // Set ip
    }
}

func (c *Cluster) Start() error {
    // ...

    // Set ip for each node
    for _, node := range c.Nodes {
        node.IP = c.IP
    }

    // ...
}