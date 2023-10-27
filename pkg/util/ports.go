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
	"net"

	"github.com/docker/go-connections/nat"
)

// GetFreePort tries to fetch an open port from the OS-Kernel
func GetFreePort() (int, error) {
	tcpAddress, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, fmt.Errorf("failed to resolve address 'localhost:0': %w", err)
	}

	tcpListener, err := net.ListenTCP("tcp", tcpAddress)
	if err != nil {
		return 0, fmt.Errorf("failed to create tcp listener: %w", err)
	}
	defer tcpListener.Close()

	return tcpListener.Addr().(*net.TCPAddr).Port, nil
}

var equalHostIPs = map[string]interface{}{
	"":          nil,
	"127.0.0.1": nil,
	"0.0.0.0":   nil,
	"localhost": nil,
}

func IsPortBindingEqual(a, b nat.PortBinding) bool {
	if a.HostPort == b.HostPort {
		if _, ok := equalHostIPs[a.HostIP]; ok {
			if _, ok := equalHostIPs[b.HostIP]; ok {
				return true
			}
		}
	}
	return false
}
