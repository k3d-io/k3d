/*
Copyright © 2022 The k3d Author(s)

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
	"context"
	"os"
	"testing"

	"github.com/go-test/deep"
)

func Test_findRuntimeImage(T *testing.T) {
	imageRegistry := []string{
		"busybox:latest",
		"busybox:version",
		"registry/one:version",
		"registry/one:latest",
		"registry/one/two:version",
		"registry/one/two:latest",
		"registry/one/two/three/four:version",
		"registry/one/two/three/four:latest",
		"registry:1234/one:version",
		"registry:1234/one:latest",
		"registry:1234/one/two:version",
		"registry:1234/one/two:latest",
		"registry:1234/one/two/three/four:version",
		"registry:1234/one/two/three/four:latest",
		"one/two:latest",
	}

	tests := map[string]struct {
		expectedImageName       string
		expectedFound           bool
		givenRequestedImageName string
	}{
		"registry image tag": {
			expectedImageName:       "registry/one/two:latest",
			expectedFound:           true,
			givenRequestedImageName: "registry/one/two",
		},
		"registry image tag with version": {
			expectedImageName:       "registry/one/two:version",
			expectedFound:           true,
			givenRequestedImageName: "registry/one/two:version",
		},
		"registry image tag with short path": {
			expectedImageName:       "registry/one:latest",
			expectedFound:           true,
			givenRequestedImageName: "registry/one",
		},
		"registry image tag with short path and specific version": {
			expectedImageName:       "registry/one:version",
			expectedFound:           true,
			givenRequestedImageName: "registry/one:version",
		},
		"registry image tag with short path and registry port": {
			expectedImageName:       "registry:1234/one:latest",
			expectedFound:           true,
			givenRequestedImageName: "registry:1234/one",
		},
		"registry image tag with short path and specific version, registry port": {
			expectedImageName:       "registry:1234/one:version",
			expectedFound:           true,
			givenRequestedImageName: "registry:1234/one:version",
		},
		"registry image tag with long path": {
			expectedImageName:       "registry/one/two/three/four:latest",
			expectedFound:           true,
			givenRequestedImageName: "registry/one/two/three/four",
		},
		"registry image tag with long path and specific version": {
			expectedImageName:       "registry/one/two/three/four:version",
			expectedFound:           true,
			givenRequestedImageName: "registry/one/two/three/four:version",
		},
		"registry image tag with long path and repository port": {
			expectedImageName:       "registry:1234/one/two/three/four:latest",
			expectedFound:           true,
			givenRequestedImageName: "registry:1234/one/two/three/four",
		},
		"registry image tag with long path, specific version and repository port": {
			expectedImageName:       "registry:1234/one/two/three/four:version",
			expectedFound:           true,
			givenRequestedImageName: "registry:1234/one/two/three/four:version",
		},
		"plain library image tag": {
			expectedImageName:       "busybox:latest",
			expectedFound:           true,
			givenRequestedImageName: "busybox",
		},
		"plain library image tag with version": {
			expectedImageName:       "busybox:latest",
			expectedFound:           true,
			givenRequestedImageName: "busybox:latest",
		},
		"library image tag": {
			expectedImageName:       "busybox:latest",
			expectedFound:           true,
			givenRequestedImageName: "library/busybox",
		},
		"library image tag with latest version": {
			expectedImageName:       "busybox:latest",
			expectedFound:           true,
			givenRequestedImageName: "library/busybox:latest",
		},
		"library image tag with specific version": {
			expectedImageName:       "busybox:latest",
			expectedFound:           true,
			givenRequestedImageName: "library/busybox:latest",
		},
		"library image tag with repository": {
			expectedImageName:       "busybox:latest",
			expectedFound:           true,
			givenRequestedImageName: "docker.io/library/busybox",
		},
		"library image tag with repository and latest version": {
			expectedImageName:       "busybox:latest",
			expectedFound:           true,
			givenRequestedImageName: "docker.io/library/busybox:latest",
		},
		"library image tag with repository and specific version": {
			expectedImageName:       "busybox:version",
			expectedFound:           true,
			givenRequestedImageName: "docker.io/library/busybox:version",
		},
		"docker.io is used as registry": {
			expectedImageName:       "one/two:latest",
			expectedFound:           true,
			givenRequestedImageName: "docker.io/one/two:latest",
		},
		"docker.io is used as registry, no version tag": {
			expectedImageName:       "one/two:latest",
			expectedFound:           true,
			givenRequestedImageName: "docker.io/one/two",
		},
		"unknown image": {
			expectedFound:           false,
			givenRequestedImageName: "unknown",
		},
		"unknown with version": {
			expectedFound:           false,
			givenRequestedImageName: "unknown:latest",
		},
		"unknown with repository": {
			expectedFound:           false,
			givenRequestedImageName: "docker.io/unknown",
		},
		"unknown with repository and version": {
			expectedFound:           false,
			givenRequestedImageName: "docker.io/unknown:tag",
		},
	}

	for name, tt := range tests {
		T.Run(name, func(t *testing.T) {
			actualImageName, actualFound := findRuntimeImage(tt.givenRequestedImageName, imageRegistry)

			if tt.expectedFound != actualFound {
				t.Errorf("The image '%s' should not have been found.", tt.givenRequestedImageName)
			}
			if tt.expectedImageName != actualImageName {
				t.Errorf("The image '%s' was found, but '%s' was expected.", actualImageName, tt.expectedImageName)
			}
		})
	}
}

func Test_findImages(t *testing.T) {
	// given
	tarImage, err := os.CreateTemp("", "images.tgz")
	if err != nil {
		t.Fatal("Failed to create temporary file")
	}
	defer os.Remove(tarImage.Name())

	tarImages := []string{tarImage.Name()}
	runtimeImages := []string{
		"alpine:version",
		"busybox:latest",
	}
	runtime := &FakeRuntimeImageGetter{runtimeImages: runtimeImages}

	requestedImages := append(runtimeImages, tarImages...)

	// when
	foundRuntimeImages, foundTarImages, err := findImages(context.Background(), runtime, requestedImages)

	// then
	if err != nil {
		t.Errorf("Got unexpected error %v", err)
	}

	if diff := deep.Equal(foundRuntimeImages, runtimeImages); diff != nil {
		t.Errorf("Found runtime images\n%+v\ndoes not match expected runtime images\n%+v\nDiff:\n%+v", foundRuntimeImages, runtimeImages, diff)
	}
	if diff := deep.Equal(foundTarImages, tarImages); diff != nil {
		t.Errorf("Found tar images\n%+v\ndoes not match expected tar images\n%+v\nDiff:\n%+v", foundTarImages, runtimeImages, diff)
	}
}

type FakeRuntimeImageGetter struct {
	runtimeImages []string
}

func (f *FakeRuntimeImageGetter) GetImages(_ context.Context) ([]string, error) {
	return f.runtimeImages, nil
}
