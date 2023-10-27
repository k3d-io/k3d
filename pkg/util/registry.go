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
package util

import (
	"fmt"
	"regexp"

	"github.com/docker/go-connections/nat"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

var registryRefRegexp = regexp.MustCompile(`^(?P<protocol>http:\/\/|https:\/\/)?(?P<hostref>(?P<hostip>\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})|(?P<hostname>[a-zA-Z\-\.0-9]+)){1}?((:)(?P<internalport>\d{1,5}))?((:)(?P<externalport>\d{1,5}))?$`)

// ParseRegistryRef returns a registry struct parsed from a simplified definition string
func ParseRegistryRef(registryRef string) (*k3d.Registry, error) {
	match := registryRefRegexp.FindStringSubmatch(registryRef)

	if len(match) == 0 {
		return nil, fmt.Errorf("Failed to parse registry reference %s: Must be [proto://]host[:port]", registryRef)
	}

	submatches := MapSubexpNames(registryRefRegexp.SubexpNames(), match)

	registry := &k3d.Registry{
		Host:         submatches["hostref"],
		Protocol:     submatches["protocol"],
		ExposureOpts: k3d.ExposureOpts{},
	}
	registry.ExposureOpts.Host = submatches["hostref"]

	if submatches["port"] != "" {
		registry.ExposureOpts.PortMapping = nat.PortMapping{
			Port: nat.Port(fmt.Sprintf("%s/tcp", submatches["internalport"])),
			Binding: nat.PortBinding{
				HostPort: submatches["externalport"],
			},
		}
	}
	return registry, nil
}
