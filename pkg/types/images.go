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
package types

import (
	"fmt"
	"os"

	"github.com/rancher/k3d/v4/version"
	log "github.com/sirupsen/logrus"
)

// DefaultK3sImageRepo specifies the default image repository for the used k3s image
const DefaultK3sImageRepo = "docker.io/rancher/k3s"

// DefaultLBImageRepo defines the default cluster load balancer image
const DefaultLBImageRepo = "docker.io/rancher/k3d-proxy"

// DefaultToolsImageRepo defines the default image used for the tools container
const DefaultToolsImageRepo = "docker.io/rancher/k3d-tools"

// DefaultRegistryImageRepo defines the default image used for the k3d-managed registry
const DefaultRegistryImageRepo = "docker.io/library/registry"

// DefaultRegistryImageTag defines the default image tag used for the k3d-managed registry
const DefaultRegistryImageTag = "2"

func GetLoadbalancerImage() string {
	if img := os.Getenv("K3D_IMAGE_LOADBALANCER"); img != "" {
		log.Infof("Loadbalancer image set from env var $K3D_IMAGE_LOADBALANCER: %s", img)
		return img
	}

	return fmt.Sprintf("%s:%s", DefaultLBImageRepo, version.GetHelperImageVersion())
}

func GetToolsImage() string {
	if img := os.Getenv("K3D_IMAGE_TOOLS"); img != "" {
		log.Infof("Tools image set from env var $K3D_IMAGE_TOOLS: %s", img)
		return img
	}

	return fmt.Sprintf("%s:%s", DefaultToolsImageRepo, version.GetHelperImageVersion())
}
