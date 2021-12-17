/*
Copyright © 2020-2021 The k3d Author(s)

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package containerd

import (
	"context"
	"fmt"
	"strings"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
	k3d "github.com/rancher/k3d/v5/pkg/types"
)

func TranslateNodeToContainer(ctx context.Context, client *containerd.Client, node *k3d.Node) (*NodeInContainerd, error) {

	// container, err := client.NewContainer(
	// 	ctx,
	// 	name,
	// 	containerd.WithImage(image),
	// 	containerd.WithSnapshotter(snapshotter),
	// 	containerd.WithNewSnapshot(name+"-snapshot", image),
	// 	containerd.WithNewSpec(oci.WithImageConfig(image),
	// 		oci.WithHostname(name),
	// 		oci.WithCapabilities([]string{"CAP_NET_RAW"}),
	// 		oci.WithMounts(mounts),
	// 		oci.WithEnv(envs),
	// 		withMemory(memory)),
	// 	containerd.WithContainerLabels(labels),
	// )

	// TODO: pull image if not present or ensure presence somewhere else
	image, err := client.GetImage(ctx, node.Image)
	if err != nil {
		return nil, err
	}

	var mounts []specs.Mount
	for _, volume := range node.Volumes {
		mount := specs.Mount{
			Type: "bind",
		}
		split := strings.Split(volume, ":")

		// 1. more than 2 parts: error
		if len(split) > 3 {
			return nil, fmt.Errorf("Invalid volume definition %s (too many colons)", volume)
		}

		// 2. more than 1 part
		if len(split) > 1 {
			// 2.1: second part is destination (absolute path only)
			if strings.HasPrefix(split[1], "/") {
				mount.Destination = split[1]

				// 2.2: second part is opts already
			} else {
				mount.Options = append(mount.Options, strings.Split(split[1], ",")...)
			}
		}

		// 3. three parts: last part is opts
		if len(split) == 3 {
			mount.Options = append(mount.Options, strings.Split(split[2], ",")...)
		}

		mounts = append(mounts, mount)
	}

	container, err := generateContainer(ctx, client, node.Name,
		containerd.WithImageName(node.Image),
		containerd.WithContainerLabels(node.RuntimeLabels),
		containerd.WithNewSpec(
			oci.WithImageConfig(image),
			oci.WithPrivileged,
			oci.WithMounts(mounts),
		),
	)
	if err != nil {
		return nil, err
	}

	return &NodeInContainerd{
		Container: container,
	}, nil
}

func generateContainer(ctx context.Context, client *containerd.Client, name string, opts ...containerd.NewContainerOpts) (*containers.Container, error) {
	ctx, done, err := client.WithLease(ctx)
	if err != nil {
		return nil, err
	}
	defer done(ctx)

	container := &containers.Container{
		ID: name,
		Runtime: containers.RuntimeInfo{
			Name: client.Runtime(),
		},
	}
	for _, o := range opts {
		if err := o(ctx, client, container); err != nil {
			return nil, err
		}
	}
	return container, nil
}
