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
package client

import (
	"os"
	"strconv"

	l "github.com/rancher/k3d/v5/pkg/logger"
	"github.com/rancher/k3d/v5/pkg/runtimes"
	"github.com/rancher/k3d/v5/pkg/types/fixes"
)

// FIXME: FixCgroupV2 - to be removed when fixed upstream
func EnableCgroupV2FixIfNeeded(runtime runtimes.Runtime) {
	if _, isSet := os.LookupEnv(string(fixes.EnvFixCgroupV2)); !isSet {
		runtimeInfo, err := runtime.Info()
		if err != nil {
			l.Log().Warnf("Failed to get runtime information: %+v", err)
			return
		}
		cgroupVersion, err := strconv.Atoi(runtimeInfo.CgroupVersion)
		if err != nil {
			l.Log().Debugf("Failed to parse cgroupVersion: %+v", err)
			return
		}
		if cgroupVersion == 2 {
			l.Log().Debugf("Detected CgroupV2, enabling custom entrypoint (disable by setting %s=false)", fixes.EnvFixCgroupV2)
			if err := os.Setenv(string(fixes.EnvFixCgroupV2), "true"); err != nil {
				l.Log().Errorf("Detected CgroupsV2 but failed to enable k3d's hotfix (try `export %s=true`): %+v", fixes.EnvFixCgroupV2, err)
			}
		}
	}
}
