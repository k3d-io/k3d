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
package fixes

import (
	_ "embed"
	"os"
	"strconv"

	k3d "github.com/rancher/k3d/v5/pkg/types"
)

/* NOTE
 * This file includes types used for workarounds and hotfixes which are subject to change
 * and may disappear anytime, e.g. when the fix was included in an upstream project
 */

/*
 * Cgroupv2 fix as per https://github.com/k3s-io/k3s/pull/3237 & https://github.com/k3s-io/k3s/pull/3242
 * FIXME: FixCgroupV2 - to be removed when fixed upstream
 */

type K3DFixEnv string

const (
	EnvFixCgroupV2 K3DFixEnv = k3d.K3dEnvFixCgroupV2 // EnvFixCgroupV2 is the environment variable that k3d will check for to enable/disable the cgroupv2 workaround
	EnvFixDNS      K3DFixEnv = k3d.K3dEnvFixDNS      // EnvFixDNS is the environment variable that check for to enable/disable the application of network magic related to DNS
)

var FixEnvs []K3DFixEnv = []K3DFixEnv{
	EnvFixCgroupV2,
	EnvFixDNS,
}

//go:embed assets/k3d-entrypoint-cgroupv2.sh
var CgroupV2Entrypoint []byte

//go:embed assets/k3d-entrypoint-dns.sh
var DNSMagicEntrypoint []byte

//go:embed assets/k3d-entrypoint.sh
var K3DEntrypoint []byte

func FixEnabled(fixenv K3DFixEnv) bool {
	enabled, err := strconv.ParseBool(os.Getenv(string(fixenv)))
	if err != nil {
		return false
	}
	return enabled
}

func FixEnabledAny() bool {
	for _, fixenv := range FixEnvs {
		if FixEnabled(fixenv) {
			return true
		}
	}
	return false
}
