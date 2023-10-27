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
	"strings"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
)

// SplitFiltersFromFlag separates a flag's value from the node filter, if there is one
func SplitFiltersFromFlag(flag string) (string, []string, error) {
	/* Case 1) no filter specified */

	if !strings.Contains(flag, "@") {
		return flag, nil, nil
	}

	/* Case 2) filter indicated using '@' in flag */

	split := strings.Split(flag, "@")
	newsplit := []string{}
	buffer := ""

	for i, it := range split {
		// Case 1: There's a '\' just before the '@' sign -> Should it be escaped (aka be a literal '@')?
		if strings.HasSuffix(it, "\\") && i != len(split)-1 {
			// Case 1.1: Escaped backslash
			if strings.HasSuffix(it, "\\\\") {
				it = strings.TrimSuffix(it, "\\")
				l.Log().Warnf("The part '%s' of the flag input '%s' ends with a double backslash, so we assume you want to escape the backslash before the '@'. That's the only time we do this.", it, flag)
			} else {
				// Case 1.2: Unescaped backslash -> Escaping the '@' -> remove suffix and append it to buffer, followed by the escaped @ sign
				l.Log().Tracef("Item '%s' just before an '@' ends with '\\', so we assume it's escaping a literal '@'", it)
				buffer += strings.TrimSuffix(it, "\\") + "@"
				continue
			}
		}
		// Case 2: There's no '\': append item to buffer, save it to new slice, empty buffer and continue
		newsplit = append(newsplit, buffer+it)
		buffer = ""
		continue
	}

	// max number of pieces after split = 2 (only one @ allowed in flag)
	if len(newsplit) > 2 {
		return "", nil, fmt.Errorf("Invalid flag '%s': only one unescaped '@' allowed for node filter(s) (Escape literal '@' with '\\')", flag)
	}

	// trailing or leading '@'
	if len(newsplit) < 2 {
		return "", nil, fmt.Errorf("Invalid flag '%s' includes unescaped '@' but is missing a node filter (Escape literal '@' with '\\')", flag)
	}

	return newsplit[0], strings.Split(newsplit[1], ";"), nil
}
