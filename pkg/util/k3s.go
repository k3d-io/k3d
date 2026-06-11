package util

import (
    // ... existing imports ...
    "github.com/k3d-io/k3d/pkg/types"
    // ... existing imports ...
)

func getK3sArgs(config types.ClusterCreateConfig) []string {
    args := []string{}
    // ... existing code ...

    if config.TLSSAN != "" {
        args = append(args, "--tls-san", config.TLSSAN)
    }

    // ... existing code ...
    return args
}