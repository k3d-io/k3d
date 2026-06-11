package main

import (
    // ... existing imports ...
    "github.com/k3d-io/k3d/pkg/types"
    // ... existing imports ...
)

func createCluster(config types.ClusterCreateConfig) error {
    // ... existing code ...

    k3sArgs := config.ToK3sArgs()
    // ... existing code ...

    return nil
}