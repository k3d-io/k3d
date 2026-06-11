package cmd

import (
    "fmt"
    "github.com/k3d-io/k3d/v5/pkg/types"
    "github.com/spf13/cobra"
)

func (c *clusterCreateCmd) RunE(cmd *cobra.Command, args []string) error {
    // ...

    cluster := types.Cluster{
        Name:     c.name,
        Servers:  c.servers,
        Agents:   c.agents,
        Port:     c.port,
        APIPort:  c.apiPort,
        IP:       c.ip, // Set ip
    }

    // ...
}