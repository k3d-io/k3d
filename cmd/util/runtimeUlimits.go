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
	"strconv"
	"strings"

	dockerunits "github.com/docker/go-units"

	"github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
)

type UlimitTypes interface {
	dockerunits.Ulimit | v1alpha5.Ulimit
}

type Ulimit[T UlimitTypes] struct {
	Values T
}

// ValidateRuntimeUlimitKey validates a given ulimit key is valid
func ValidateRuntimeUlimitKey(ulimitKey string) {
	ulimitsKeys := map[string]bool{
		"core":       true,
		"cpu":        true,
		"data":       true,
		"fsize":      true,
		"locks":      true,
		"memlock":    true,
		"msgqueue":   true,
		"nice":       true,
		"nofile":     true,
		"nproc":      true,
		"rss":        true,
		"rtprio":     true,
		"rttime":     true,
		"sigpending": true,
		"stack":      true,
	}
	keysList := make([]string, 0, len(ulimitsKeys))

	for key := range ulimitsKeys {
		keysList = append(keysList, key)
	}
	if !ulimitsKeys[ulimitKey] {
		l.Log().Fatalf("runtime ulimit \"%s\" is not valid, allowed keys are: %s", ulimitKey, strings.Join(keysList, ", "))
	}
}

func ParseRuntimeUlimit[T UlimitTypes](ulimit string) *T {
	var parsedUlimit any
	var tmpUlimit Ulimit[T]
	ulimitSplitted := strings.Split(ulimit, "=")
	if len(ulimitSplitted) != 2 {
		l.Log().Fatalf("unknown runtime-ulimit format format: %s, use format \"ulimit=soft:hard\"", ulimit)
	}
	ValidateRuntimeUlimitKey(ulimitSplitted[0])
	softHardSplitted := strings.Split(ulimitSplitted[1], ":")
	if len(softHardSplitted) != 2 {
		l.Log().Fatalf("unknown runtime-ulimit format format: %s, use format \"ulimit=soft:hard\"", ulimit)
	}
	soft, err := strconv.Atoi(softHardSplitted[0])
	if err != nil {
		l.Log().Fatalf("unknown runtime-ulimit format format: soft %s has to be int", ulimitSplitted[0])
	}
	hard, err := strconv.Atoi(softHardSplitted[1])
	if err != nil {
		l.Log().Fatalf("unknown runtime-ulimit format format: hard %s has to be int", ulimitSplitted[1])
	}

	switch any(tmpUlimit.Values).(type) {
	case dockerunits.Ulimit:
		parsedUlimit = &dockerunits.Ulimit{
			Name: ulimitSplitted[0],
			Soft: int64(soft),
			Hard: int64(hard),
		}
	case v1alpha5.Ulimit:
		parsedUlimit = &v1alpha5.Ulimit{
			Name: ulimitSplitted[0],
			Soft: int64(soft),
			Hard: int64(hard),
		}
	default:
		l.Log().Fatalf("Unsupported UlimitTypes, supported types are: dockerunits.Ulimit or v1alpha5.Ulimit")
	}

	return parsedUlimit.(*T)
}
