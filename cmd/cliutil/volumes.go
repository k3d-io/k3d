/*
Copyright © 2019 Thorsten Klein <iwilltry42@gmail.com>

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
package cliutil

import (
	"fmt"
	"os"
	"strings"
)

// ValidateVolumeFlag checks, if the source of volume mounts exists and if the destination is an absolute path
func ValidateVolumeFlag(volumes []string) error {

	for _, volume := range volumes {
		src := ""
		dest := ""

		if strings.Contains(volume, "@") {
			split := strings.Split(volume, "@")
			if len(split) != 2 {
				return fmt.Errorf("Invalid volume flag '%s'", volume)
			}

			// validate the passed parameter
			split := strings.Split(volume, ":")
			if len(split) < 1 || len(split) > 2 {
				return fmt.Errorf("Invalid volume mount '%s'", volume)
			}
			if len(split) == 1 {
				dest = split[0]
			} else {
				src = split[0]
				dest = split[1]
			}

			// verify that the source exists
			if src != "" {
				if _, err := os.Stat(src); err != nil {
					return fmt.Errorf("Failed to stat file/dir that you're trying to mount: '%s' in '%s'", src, volume)
				}
			}

			// verify that the destination is an absolute path
			if !strings.HasPrefix(dest, "/") {
				return fmt.Errorf("Volume mount destination doesn't appear to be an absolute path: '%s' in '%s'", dest, volume)
			}
		}

	}
	return nil
}
