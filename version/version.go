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
package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/k3d-io/k3d/v5/pkg/types/k3s"
)

// Version is the string that contains version
var Version string

// HelperVersionOverride decouples the k3d helper image versions from the main version, if needed
var HelperVersionOverride string

// K3sVersion should contain the latest version tag of k3s (hardcoded at build time)
var K3sVersion = "v1.32.5+k3s1"

type httpClient struct {
	client  *http.Client
	baseURL string
}

// GetVersion returns the version for cli, it gets it from "git describe --tags" or returns "dev" when doing simple go build
func GetVersion() string {
	if len(Version) == 0 {
		return "v5-dev"
	}
	return Version
}

// GetK3sVersion returns the version string for K3s
func GetK3sVersion(channel string) (string, error) {
	if channel != "" {
		version, err := newHttpClient(k3s.K3sChannelServerURL).fetchLatestK3sVersion(channel)
		if err != nil {
			return "", fmt.Errorf("error getting K3s version for channel %s: %w", channel, err)
		}
		return version, nil
	}
	return K3sVersion, nil
}

// fetchLatestK3sVersion tries to fetch the latest version of k3s from DockerHub
func (c *httpClient) fetchLatestK3sVersion(channel string) (string, error) {
	request, err := http.NewRequest(http.MethodGet, c.baseURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.client.Do(request)
	if err != nil {
		return "", err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(fmt.Sprintf("call made to '%s' failed with status code '%d'", c.baseURL, resp.StatusCode))
	}

	out := k3s.ChannelServerResponse{}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading channelserver response body: %w", err)
	}

	if err = json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("error unmarshalling channelserver response: %w", err)
	}

	for _, c := range out.Channels {
		if c.Name == channel {
			return strings.ReplaceAll(c.Latest, "+", "-"), err
		}
	}

	return "", fmt.Errorf("no latest version found for channel %s (%s)", channel, c.baseURL)
}

func newHttpClient(baseURL string) *httpClient {
	return &httpClient{
		client:  &http.Client{},
		baseURL: baseURL,
	}
}
