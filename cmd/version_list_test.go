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
package cmd

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/require"
)

type versionTestTransport func(*http.Request) (*http.Response, error)

func (f versionTestTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestVersionListRepoSortBySemver(t *testing.T) {
	dockerConfig := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dockerConfig, "config.json"), []byte("{}"), 0600))
	t.Setenv("DOCKER_CONFIG", dockerConfig)

	originalTransport := remote.DefaultTransport
	t.Cleanup(func() { remote.DefaultTransport = originalTransport })
	remote.DefaultTransport = versionTestTransport(func(r *http.Request) (*http.Response, error) {
		var body string
		switch r.URL.Path {
		case "/v2/":
			body = "{}"
		case "/v2/rancher/k3s/tags/list":
			body = `{"name":"rancher/k3s","tags":["v1.29.9-k3s1","v1.29.10-k3s1","v1.29.2-k3s1"]}`
		default:
			t.Errorf("unexpected registry request: %s", r.URL)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})

	const prefix = "docker.io/rancher/k3s:"
	const earlier = "v1.29.2-k3s1"
	const later = "v1.29.9-k3s1"
	const latest = "v1.29.10-k3s1"

	for _, tt := range []struct {
		name string
		args []string
		want []string
	}{
		{"raw/desc", []string{"--output", "raw", "--sort", "desc"}, []string{latest, later, earlier}},
		{"repo/desc", []string{"--output", "repo", "--sort", "desc"}, []string{prefix + latest, prefix + later, prefix + earlier}},
		{"repo/asc", []string{"--output", "repo", "--sort", "asc"}, []string{prefix + earlier, prefix + later, prefix + latest}},
		{"repo/off", []string{"--output", "repo", "--sort", "off"}, []string{prefix + later, prefix + latest, prefix + earlier}},
		{"repo/limit", []string{"--output", "repo", "--limit", "1"}, []string{prefix + latest}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reader, writer, err := os.Pipe()
			require.NoError(t, err)
			defer reader.Close()

			stdout := os.Stdout
			os.Stdout = writer
			defer func() { os.Stdout = stdout }()

			command := NewCmdVersionLs()
			command.SetArgs(append([]string{"k3s"}, tt.args...))
			err = command.Execute()
			require.NoError(t, writer.Close())
			require.NoError(t, err)

			output, err := io.ReadAll(reader)
			require.NoError(t, err)
			require.Equal(t, tt.want, strings.Fields(string(output)))
		})
	}
}
