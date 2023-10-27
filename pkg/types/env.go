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

// k3d config environment variables for options that don't have a place in the config file or CLI
const (
	// Log config
	K3dEnvLogNodeWaitLogs = "K3D_LOG_NODE_WAIT_LOGS"

	// Images
	K3dEnvImageLoadbalancer = "K3D_IMAGE_LOADBALANCER"
	K3dEnvImageTools        = "K3D_IMAGE_TOOLS"
	K3dEnvImageHelperTag    = "K3D_HELPER_IMAGE_TAG"

	// Debug options
	K3dEnvDebugCorednsRetries       = "K3D_DEBUG_COREDNS_RETRIES"
	K3dEnvDebugDisableDockerInit    = "K3D_DEBUG_DISABLE_DOCKER_INIT"
	K3dEnvDebugNodeWaitBackOffLimit = "K3D_DEBUG_NODE_WAIT_BACKOFF_LIMIT"

	// Fixes
	K3dEnvFixCgroupV2 = "K3D_FIX_CGROUPV2"
	K3dEnvFixDNS      = "K3D_FIX_DNS"
	K3dEnvFixMounts   = "K3D_FIX_MOUNTS"
)
