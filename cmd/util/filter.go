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
	"fmt"
	"strings"
)

// SplitFiltersFromFlag separates a flag's value from the node filter, if there is one
func SplitFiltersFromFlag(flag string) (string, []string, error) {

	/* Case 1) no filter specified */

	if !strings.Contains(flag, "@") {
		return flag, nil, nil
	}

	/* Case 2) filter indicated using '@' in flag */

	split := strings.Split(flag, "@")

	// max number of pieces after split = 2 (only one @ allowed in flag)
	if len(split) > 2 {
		return "", nil, fmt.Errorf("Invalid flag '%s': only one '@' for node filter allowed", flag)
	}

	// trailing or leading '@'
	if len(split) < 2 {
		return "", nil, fmt.Errorf("Invalid flag '%s' includes '@' but is missing either an object or a filter", flag)
	}

	return split[0], strings.Split(split[1], ";"), nil

}
