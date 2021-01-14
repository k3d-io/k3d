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

package util

import (
	"net"

	log "github.com/sirupsen/logrus"
)

// GetFreePort tries to fetch an open port from the OS-Kernel
func GetFreePort() (int, error) {
	tcpAddress, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		log.Errorln("Failed to resolve address")
		return 0, err
	}

	tcpListener, err := net.ListenTCP("tcp", tcpAddress)
	if err != nil {
		log.Errorln("Failed to create TCP Listener")
		return 0, err
	}
	defer tcpListener.Close()

	return tcpListener.Addr().(*net.TCPAddr).Port, nil
}
