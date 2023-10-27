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
package types

// TypeMeta is basically copied from https://github.com/kubernetes/apimachinery/blob/a3b564b22db316a41e94fdcffcf9995424fe924c/pkg/apis/meta/v1/types.go#L36-L56
type TypeMeta struct {
	Kind       string `mapstructure:"kind,omitempty" json:"kind,omitempty"`
	APIVersion string `mapstructure:"apiVersion,omitempty" json:"apiVersion,omitempty"`
}

// ObjectMeta got its name from the Kubernetes counterpart.
type ObjectMeta struct {
	Name string `mapstructure:"name,omitempty" json:"name,omitempty"`
}

// Config interface.
type Config interface {
	GetKind() string
	GetAPIVersion() string
}

// Default Targets for NodeFilters
var (
	DefaultTargetsNodefiltersPortMappings = []string{"servers:*:proxy", "agents:*:proxy"}
)
