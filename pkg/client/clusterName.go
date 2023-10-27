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
package client

import (
	"fmt"
	"regexp"

	"github.com/k3d-io/k3d/v5/pkg/types"
)

// HostnameRegexp as per RFC 1123
var HostnameRegexp = regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)

// CheckName ensures that a cluster name is also a valid host name according to RFC 1123.
// We further restrict the length of the cluster name to maximum 'clusterNameMaxSize'
// so that we can construct the host names based on the cluster name, and still stay
// within the 64 characters limit.
func CheckName(name string) error {
	if err := ValidateHostname(name); err != nil {
		return fmt.Errorf("Invalid cluster name. %+v", ValidateHostname(name))
	}
	if len(name) > types.DefaultClusterNameMaxLength {
		return fmt.Errorf("Cluster name must be <= %d characters, but has %d", types.DefaultClusterNameMaxLength, len(name))
	}
	return nil
}

// ValidateHostname ensures that a cluster name is also a valid host name according to RFC 1123.
func ValidateHostname(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("No name provided")
	}

	if !HostnameRegexp.Match([]byte(name)) {
		return fmt.Errorf("name %s is not a valid hostname as per RFC 1123", name)
	}

	return nil
}
