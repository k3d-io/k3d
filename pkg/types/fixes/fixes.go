/*
Copyright © 2020 The k3d Author(s)

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
)

/* NOTE
 * This file includes types used for workarounds and hotfixes which are subject to change
 * and may disappear anytime, e.g. when the fix was included in an upstream project
 */

/*
 * Cgroupv2 fix as per https://github.com/k3s-io/k3s/pull/3237 & https://github.com/k3s-io/k3s/pull/3242
 * FIXME: FixCgroupV2 - to be removed when fixed upstream
 */

// EnvFixCgroupV2 is the environment variable that k3d will check for to enable/disable the cgroupv2 workaround
const EnvFixCgroupV2 = "K3D_FIX_CGROUPV2"

//go:embed assets/cgroupv2-entrypoint.sh
var CgroupV2Entrypoint []byte

func FixCgroupV2Enabled() bool {
	enabled, err := strconv.ParseBool(os.Getenv(EnvFixCgroupV2))
	if err != nil {
		return false
	}
	return enabled
}
