/*
Copyright © 2020-2021 The k3d Author(s)

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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strings"

	l "github.com/rancher/k3d/v4/pkg/logger"
)

// ReleaseAsset maps a single asset in a GitHub Release
type ReleaseAsset struct {
	Name        string `json:"name"`
	DownloadUrl string `json:"browser_download_url"`
}

// ReleaseResponse maps GitHub Release APIs response
type ReleaseResponse struct {
	Tag    string         `json:"tag_name"`
	Assets []ReleaseAsset `json:"assets"`
}

// findCompatiblePackageUrl looks for assets compatible with the current system
// Return the first compatible asset if found, else nil
func (r *ReleaseResponse) findCompatibleAsset() *ReleaseAsset {
	expectedSuffix := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)

	for _, asset := range r.Assets {
		if strings.HasSuffix(asset.DownloadUrl, expectedSuffix) {
			return &asset
		}
	}

	return nil
}

// download downloads the asset in the given filepath
func (ra *ReleaseAsset) download(destinationPath string) error {
	response, err := http.Get(ra.DownloadUrl)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return errors.New("download received non 200 status code")
	}

	// Create empty file
	file, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write response to file
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	return nil
}

func pluginUrl(repo, tag string) string {
	url := "https://api.github.com/repos/%s/releases/tags/%s"

	if tag == "latest" {
		url = "https://api.github.com/repos/%s/releases/%s"
	}
	return fmt.Sprintf(url, repo, tag)
}

// fetchRelease fetches the latest release of a GitHub Release.
// githubRepository has to be formatted as username/repository.
func fetchRelease(githubRepository, tag string) (*ReleaseResponse, error) {
	var apiUrl = pluginUrl(githubRepository, tag)

	response, err := http.Get(apiUrl)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	release := &ReleaseResponse{}
	err = json.NewDecoder(response.Body).Decode(release)
	if err != nil {
		return nil, err
	}

	return release, nil
}

// DownloadPlugin downloads the tag release of
// githubRepository's repo and save it in filepath
func DownloadPlugin(githubRepository, tag, pluginPath string) error {
	// Fetch info about the specified release
	release, err := fetchRelease(githubRepository, tag)
	if err != nil {
		l.Log().Error(err)
		return errors.New("error while fetching releases")
	}

	// Make sure response is not empty
	// An empty response means that the given release does not exist
	// Reflect is used since release is a pointer to a struct
	if reflect.DeepEqual(release, &ReleaseResponse{}) {
		return errors.New("unable to fetch specified release tag. Make sure it exists")
	}

	// Get the URL to download the plugin
	asset := release.findCompatibleAsset()
	if asset == nil {
		return errors.New("plugin not found. Make sure it exists")
	}

	// Download the plugin
	err = asset.download(pluginPath)
	if err != nil {
		return err
	}

	return nil
}
