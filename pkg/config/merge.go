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

package config

import (
	"fmt"

	"github.com/imdario/mergo"
	conf "github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
)

// MergeSimple merges two simple configuration files with the values of the destination one having priority
func MergeSimple(dest, src conf.SimpleConfig) (*conf.SimpleConfig, error) {
	l.Log().Debugf("Merging %+v into %+v", src, dest)

	if err := mergo.Merge(&dest, src); err != nil {
		return nil, fmt.Errorf("failed to merge configs: %w", err)
	}

	return &dest, nil
}
