package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

func importWithPull(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster, images []string) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(images)*len(cluster.Nodes))

	for _, node := range cluster.Nodes {
		if node.Role != k3d.ServerRole && node.Role != k3d.AgentRole {
			continue
		}
		for _, img := range images {
			wg.Add(1)
			go func(n *k3d.Node, imageRef string) {
				defer wg.Done()

				// Use ctr; fully-qualified refs are safest
				cmd := []string{"crictl", "pull", imageRef}
				if err := runtime.ExecInNode(ctx, n, cmd); err != nil {
					errCh <- fmt.Errorf("node %s: pull %s failed: %w", n.Name, imageRef, err)
				}
			}(node, img)
		}
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for e := range errCh {
		errs = append(errs, e)
	}
	if len(errs) > 0 {
		return fmt.Errorf("pull-based import had %d error(s): %v", len(errs), errs)
	}
	return nil
}
