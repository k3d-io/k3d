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
	"io"
	"os"
	"strings"
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

func Test_prepareImagesForNodeImport(t *testing.T) {
	// the "*" is required so CreateTemp's random suffix lands before ".tar", not after - otherwise the
	// file wouldn't actually have a ".tar" suffix, which tarExtension (and k3s) checks for
	tarImage, err := os.CreateTemp("", "images-*.tar")
	if err != nil {
		t.Fatal("Failed to create temporary file")
	}
	defer os.Remove(tarImage.Name())

	t.Run("requested image is neither a file nor known to the runtime", func(t *testing.T) {
		runtime := &FakeRuntimeImageExporter{runtimeImages: []string{"busybox:latest"}}

		_, _, cleanup, err := prepareImagesForNodeImport(context.Background(), runtime, []string{"doesnotexist:v1"})
		if cleanup != nil {
			defer cleanup()
		}

		if err == nil {
			t.Fatal("expected an error for an image that is neither a file nor present in the runtime, got none")
		}
		if !strings.Contains(err.Error(), "doesnotexist:v1") {
			t.Errorf("expected error to mention the missing image, got: %v", err)
		}
		if runtime.getImageStreamCalls != 0 {
			t.Errorf("expected no image export attempt when a requested image can't be resolved, got %d", runtime.getImageStreamCalls)
		}
	})

	t.Run("requested tarball has no k3s-supported extension", func(t *testing.T) {
		unsupported, err := os.CreateTemp("", "images-*.whatever")
		if err != nil {
			t.Fatal("Failed to create temporary file")
		}
		defer os.Remove(unsupported.Name())

		runtime := &FakeRuntimeImageExporter{}

		_, _, cleanup, err := prepareImagesForNodeImport(context.Background(), runtime, []string{unsupported.Name()})
		if cleanup != nil {
			defer cleanup()
		}

		if err == nil {
			t.Fatal("expected an error for a tarball with an unsupported extension, got none")
		}
		if !strings.Contains(err.Error(), unsupported.Name()) {
			t.Errorf("expected error to mention the offending tarball, got: %v", err)
		}
	})

	t.Run("mix of tarball and runtime image exports the runtime image exactly once", func(t *testing.T) {
		runtime := &FakeRuntimeImageExporter{
			runtimeImages: []string{"busybox:latest"},
			streamContent: "fake-tar-content",
		}

		imageDirStage, tarballs, cleanup, err := prepareImagesForNodeImport(context.Background(), runtime, []string{tarImage.Name(), "busybox"})
		if err != nil {
			t.Fatalf("got unexpected error: %v", err)
		}
		defer cleanup()

		if runtime.getImageStreamCalls != 1 {
			t.Errorf("expected exactly one GetImageStream call (one export for all runtime images), got %d", runtime.getImageStreamCalls)
		}
		if len(tarballs) != 2 {
			t.Fatalf("expected 2 tarballs (the given one plus the exported one), got %d: %v", len(tarballs), tarballs)
		}
		if tarballs[0] != tarImage.Name() {
			t.Errorf("expected the first tarball to be the one passed in, got %s", tarballs[0])
		}
		if _, err := os.Stat(tarballs[1]); err != nil {
			t.Errorf("expected exported tarball '%s' to exist: %v", tarballs[1], err)
		}
		if got, err := os.ReadFile(tarballs[1]); err != nil || string(got) != runtime.streamContent {
			t.Errorf("expected exported tarball to contain the streamed content, got %q (err: %v)", got, err)
		}
		if _, err := os.Stat(imageDirStage); err != nil {
			t.Errorf("expected staged image directory '%s' to exist: %v", imageDirStage, err)
		}

		cleanup()
		if _, err := os.Stat(imageDirStage); !os.IsNotExist(err) {
			t.Errorf("expected cleanup to remove the staging directory '%s'", imageDirStage)
		}
	})
}

type FakeRuntimeImageExporter struct {
	runtimeImages       []string
	streamContent       string
	getImageStreamCalls int
}

func (f *FakeRuntimeImageExporter) GetImages(_ context.Context) ([]string, error) {
	return f.runtimeImages, nil
}

func (f *FakeRuntimeImageExporter) GetImageStream(_ context.Context, _ []string) (io.ReadCloser, error) {
	f.getImageStreamCalls++
	return io.NopCloser(strings.NewReader(f.streamContent)), nil
}

func Test_tarExtension(t *testing.T) {
	tests := map[string]string{
		"/path/to/image.tar":      ".tar",
		"/path/to/image.tar.gz":   ".tar.gz",
		"/path/to/image.tgz":      ".tgz",
		"/path/to/IMAGE.TAR.GZ":   ".tar.gz", // k3s matches case-insensitively
		"/path/to/image.tar.bz2":  ".tar.bz2",
		"/path/to/image.tbz":      ".tbz",
		"/path/to/image.tar.zst":  ".tar.zst",
		"/path/to/image.tzst":     ".tzst",
		"/path/to/image.tar.lz4":  ".tar.lz4",
		"/path/to/image.whatever": "",
		"/path/to/image":          "",
	}

	for path, want := range tests {
		t.Run(path, func(t *testing.T) {
			if got := tarExtension(path); got != want {
				t.Errorf("tarExtension(%q) = %q, want %q", path, got, want)
			}
		})
	}
}
