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
	"os"
	"time"

	"github.com/goodhosts/hostsfile"
	"github.com/k3d-io/k3d/v5/pkg/actions"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

func NewHostAliasesInjectEtcHostsAction(runtime runtimes.Runtime, hostAliases []k3d.HostAlias) actions.RewriteFileAction {
	return actions.RewriteFileAction{
		Runtime:     runtime,
		Path:        "/etc/hosts",
		Mode:        0644,
		Description: "Adding HostAliases to /etc/hosts in nodes",
		Opts: actions.RewriteFileActionOpts{
			NoCopy: true,
		},
		RewriteFunc: func(input []byte) ([]byte, error) {
			tmpHosts, err := os.CreateTemp("", "k3d-hostsfile-*")
			if err != nil {
				return nil, fmt.Errorf("error creating temp hosts file: %w", err)
			}
			if err := os.WriteFile(tmpHosts.Name(), input, 0777); err != nil {
				return nil, fmt.Errorf("error writing to temp hosts file: %w", err)
			}

			hFile, err := hostsfile.NewCustomHosts(tmpHosts.Name())
			if err != nil {
				return nil, fmt.Errorf("error reading temp hosts file: %w", err)
			}

			for _, hostAlias := range hostAliases {
				if err := hFile.Add(hostAlias.IP, hostAlias.Hostnames...); err != nil {
					return nil, fmt.Errorf("error adding hosts file entry for %s:%s: %w", hostAlias.IP, hostAlias.Hostnames, err)
				}
			}

			hFile.Clean()

			if err := hFile.Flush(); err != nil {
				return nil, fmt.Errorf("error flushing hosts file: %w", err)
			}

			time.Sleep(time.Second)

			return os.ReadFile(tmpHosts.Name())
		},
	}
}
