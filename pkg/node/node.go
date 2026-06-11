package node

import (
    "github.com/k3d-io/k3d/v5/pkg/types"
)

type Node struct {
    // ...
    IP string // Add ip field
}

func NewNode(name string, ip string) *Node {
    return &Node{
        // ...
        IP: ip, // Set ip
    }
}