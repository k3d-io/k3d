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
package fixes

import (
	_ "embed"
	"os"
	"strconv"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

/* NOTE
 * This file includes types used for workarounds and hotfixes which are subject to change
 * and may disappear anytime, e.g. when the fix was included in an upstream project
 */

/*
 * Cgroupv2 fix as per https://github.com/k3s-io/k3s/pull/3237 & https://github.com/k3s-io/k3s/pull/3242
 * Since we're NOT running K3s as PID 1 (using init), we still need our fix even though we have the fix upstream https://github.com/k3s-io/k3s/pull/4086#issuecomment-931639392
 */

type K3DFixEnv string

const (
	EnvFixCgroupV2 K3DFixEnv = k3d.K3dEnvFixCgroupV2 // EnvFixCgroupV2 is the environment variable that k3d will check for to enable/disable the cgroupv2 workaround
	EnvFixDNS      K3DFixEnv = k3d.K3dEnvFixDNS      // EnvFixDNS is the environment variable that k3d will check for to enable/disable the application of network magic related to DNS
	EnvFixMounts   K3DFixEnv = k3d.K3dEnvFixMounts   // EnvFixMounts is the environment variable that k3d will check for to enable/disable the fixing of mountpoints
)

var FixEnvs []K3DFixEnv = []K3DFixEnv{
	EnvFixCgroupV2,
	EnvFixDNS,
	EnvFixMounts,
}

//go:embed assets/k3d-entrypoint-cgroupv2.sh
var CgroupV2Entrypoint []byte

//go:embed assets/k3d-entrypoint-dns.sh
var DNSMagicEntrypoint []byte

//go:embed assets/k3d-entrypoint-mounts.sh
var MountsEntrypoint []byte

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

var EnabledFixes map[K3DFixEnv]bool
var AnyFixEnabled bool

var fixNeeded = map[K3DFixEnv]func(runtime runtimes.Runtime) bool{
	EnvFixCgroupV2: func(runtime runtimes.Runtime) bool {
		runtimeInfo, err := runtime.Info()
		if err != nil {
			l.Log().Warnf("Failed to get runtime information: %+v", err)
			return false
		}
		cgroupVersion, err := strconv.Atoi(runtimeInfo.CgroupVersion)
		if err != nil {
			l.Log().Debugf("Failed to parse cgroupVersion: %+v", err)
			return false
		}
		l.Log().Debugf("[autofix cgroupsv2] cgroupVersion: %d", cgroupVersion)
		return cgroupVersion == 2
	},
	EnvFixDNS: func(_ runtimes.Runtime) bool {
		return true
	},
	EnvFixMounts: func(_ runtimes.Runtime) bool {
		return true
	},
}

// GetFixes returns a map showing which fixes are enabled and a helper boolean indicating if any fixes are enabled
func GetFixes(runtime runtimes.Runtime) (map[K3DFixEnv]bool, bool) {
	if EnabledFixes == nil {
		result := make(map[K3DFixEnv]bool, len(FixEnvs))
		anyEnabled := false
		for _, fixEnv := range FixEnvs {
			enabled := false
			if v, isSet := os.LookupEnv(string(fixEnv)); !isSet {
				enabled = fixNeeded[fixEnv](runtime)
			} else {
				var err error
				enabled, err = strconv.ParseBool(v)
				if err != nil {
					enabled = false
				}
			}
			result[fixEnv] = enabled
			if enabled {
				anyEnabled = true
			}
		}
		EnabledFixes = result
		AnyFixEnabled = anyEnabled
	}
	return EnabledFixes, AnyFixEnabled
}
