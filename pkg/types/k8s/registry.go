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

package k8s

/*
 * Source: https://github.com/kubernetes/enhancements/blob/0d69f7cea6fbe73a7d70fab569c6898f5ccb7be0/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry/README.md#specification-for-localregistryhosting-v1
 * Copied over: 07.01.2020
 * Original License
 *  > Copyright 2020 The Kubernetes Authors
 *  >
 *  > Licensed under the Apache License, Version 2.0 (the "License");
 *  > you may not use this file except in compliance with the License.
 *  > You may obtain a copy of the License at
 *  >
 *  >     http://www.apache.org/licenses/LICENSE-2.0
 *  >
 *  > Unless required by applicable law or agreed to in writing, software
 *  > distributed under the License is distributed on an "AS IS" BASIS,
 *  > WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  > See the License for the specific language governing permissions and
 *  > limitations under the License.
 */

// LocalRegistryHostingV1 describes a local registry that developer tools can
// connect to. A local registry allows clients to load images into the local
// cluster by pushing to this registry.
type LocalRegistryHostingV1 struct {
	// Host documents the host (hostname and port) of the registry, as seen from
	// outside the cluster.
	//
	// This is the registry host that tools outside the cluster should push images
	// to.
	Host string `json:"host,omitempty"`

	// HostFromClusterNetwork documents the host (hostname and port) of the
	// registry, as seen from networking inside the container pods.
	//
	// This is the registry host that tools running on pods inside the cluster
	// should push images to. If not set, then tools inside the cluster should
	// assume the local registry is not available to them.
	HostFromClusterNetwork string `json:"hostFromClusterNetwork,omitempty"`

	// HostFromContainerRuntime documents the host (hostname and port) of the
	// registry, as seen from the cluster's container runtime.
	//
	// When tools apply Kubernetes objects to the cluster, this host should be
	// used for image name fields. If not set, users of this field should use the
	// value of Host instead.
	//
	// Note that it doesn't make sense semantically to define this field, but not
	// define Host or HostFromClusterNetwork. That would imply a way to pull
	// images without a way to push images.
	HostFromContainerRuntime string `json:"hostFromContainerRuntime,omitempty"`

	// Help contains a URL pointing to documentation for users on how to set
	// up and configure a local registry.
	//
	// Tools can use this to nudge users to enable the registry. When possible,
	// the writer should use as permanent a URL as possible to prevent drift
	// (e.g., a version control SHA).
	//
	// When image pushes to a registry host specified in one of the other fields
	// fail, the tool should display this help URL to the user. The help URL
	// should contain instructions on how to diagnose broken or misconfigured
	// registries.
	Help string `json:"help,omitempty"`
}

// LocalRegistryHosting defaults
const (
	LocalRegistryHostingNamespace = "kube-public"
	LocalRegistryHostingName      = "local-registry-hosting"
	LocalRegistryHostingData      = "localRegistryHosting.v1"
)
