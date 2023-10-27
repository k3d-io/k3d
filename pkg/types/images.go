/*
Copyright © 2020-2023 The k3d Author(s)

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
	"strings"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/version"
)

// DefaultK3sImageRepo specifies the default image repository for the used k3s image
const DefaultK3sImageRepo = "docker.io/rancher/k3s"

// DefaultLBImageRepo defines the default cluster load balancer image
const DefaultLBImageRepo = "ghcr.io/k3d-io/k3d-proxy"

// DefaultToolsImageRepo defines the default image used for the tools container
const DefaultToolsImageRepo = "ghcr.io/k3d-io/k3d-tools"

// DefaultRegistryImageRepo defines the default image used for the k3d-managed registry
const DefaultRegistryImageRepo = "docker.io/library/registry"

// DefaultRegistryImageTag defines the default image tag used for the k3d-managed registry
const DefaultRegistryImageTag = "2"

func GetLoadbalancerImage() string {
	if img := os.Getenv(K3dEnvImageLoadbalancer); img != "" {
		l.Log().Infof("Loadbalancer image set from env var $%s: %s", K3dEnvImageLoadbalancer, img)
		return img
	}

	return fmt.Sprintf("%s:%s", DefaultLBImageRepo, GetHelperImageVersion())
}

func GetToolsImage() string {
	if img := os.Getenv(K3dEnvImageTools); img != "" {
		l.Log().Infof("Tools image set from env var $%s: %s", K3dEnvImageTools, img)
		return img
	}

	return fmt.Sprintf("%s:%s", DefaultToolsImageRepo, GetHelperImageVersion())
}

// GetHelperImageVersion returns the CLI version or 'latest'
func GetHelperImageVersion() string {
	if tag := os.Getenv(K3dEnvImageHelperTag); tag != "" {
		l.Log().Infoln("Helper image tag set from env var")
		return tag
	}
	if len(version.HelperVersionOverride) > 0 {
		return version.HelperVersionOverride
	}
	if len(version.Version) == 0 {
		return "latest"
	}
	return strings.TrimPrefix(version.Version, "v")
}
