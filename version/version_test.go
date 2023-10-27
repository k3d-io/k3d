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
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

// GetK3sVersion returns the version string for K3s
func TestGetK3sVersion(t *testing.T) {
	searchVersion := "v1.20"
	latestRegexp := regexp.MustCompile(`v1\.20\..*`)
	v, err := GetK3sVersion(searchVersion)
	if err != nil {
		t.Fatal(err)
	}
	if !latestRegexp.Match([]byte(v)) {
		t.Fatalf("found version %s (searched: %s) does not match respective latest regexp `%s`", v, searchVersion, latestRegexp.String())
	}
}

func TestFetchLatestK3sVersion(t *testing.T) {
	type errorTestCases struct {
		description        string
		channel            string
		mockServer         *httptest.Server
		expected           string
		expectedError      bool
		constructErrString bool
		errorString        string
	}
	tests := []errorTestCases{
		{
			description: "should be able to fetch the latest k3s version successfully",
			channel:     "v1.24",
			expected:    "v1.24.6-k3s1",
			mockServer: mockServer([]byte(`{
  "data": [
    {
      "id": "stable",
      "type": "channel",
      "links": {
        "self": "https://update.k3s.io/v1-release/channels/stable"
      },
      "name": "v1.24",
      "latest": "v1.24.6+k3s1"
    },
    {
      "id": "latest",
      "type": "channel",
      "links": {
        "self": "https://update.k3s.io/v1-release/channels/latest"
      },
      "name": "latest",
      "latest": "v1.25.2+k3s1",
      "latestRegexp": ".*",
      "excludeRegexp": "^[^+]+-"
    }
  ]
}`), http.StatusOK),
		},
		{
			description: "should be able to fetch the latest k3s version successfully",
			channel:     "v1.25",
			expected:    "v1.25.2-k3s1",
			mockServer: mockServer([]byte(`{
  "data": [
    {
      "id": "stable",
      "type": "channel",
      "links": {
        "self": "https://update.k3s.io/v1-release/channels/stable"
      },
      "name": "stable",
      "latest": "v1.24.6+k3s1"
    },
    {
      "id": "latest",
      "type": "channel",
      "links": {
        "self": "https://update.k3s.io/v1-release/channels/latest"
      },
      "name": "v1.25",
      "latest": "v1.25.2+k3s1",
      "latestRegexp": ".*",
      "excludeRegexp": "^[^+]+-"
    }
  ]
}`), http.StatusOK),
		},
		{
			description:        "should error out as no latest version found for selected channel",
			channel:            "v1.25",
			expected:           "",
			expectedError:      true,
			errorString:        "no latest version found for channel v1.25 (%s)",
			constructErrString: true,
			mockServer: mockServer([]byte(`{
  "data": [
    {
      "id": "stable",
      "type": "channel",
      "links": {
        "self": "https://update.k3s.io/v1-release/channels/stable"
      },
      "name": "stable",
      "latest": "v1.24.6+k3s1"
    }
  ]
}`), http.StatusOK),
		},
		{
			description:        "should error out while fetching the latest k3s version as call made to URL returned 500",
			channel:            "v1.25",
			expected:           "",
			mockServer:         mockServer([]byte(``), http.StatusInternalServerError),
			expectedError:      true,
			constructErrString: true,
			errorString:        "call made to '%s' failed with status code '500'",
		},
		{
			description:   "should error out while fetching the latest k3s version as response returned in not understandable",
			channel:       "v1.25",
			expected:      "",
			mockServer:    mockServer([]byte(`{"data": [`), http.StatusOK),
			expectedError: true,
			errorString:   "error unmarshalling channelserver response: unexpected end of JSON input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			client := httpClient{client: tt.mockServer.Client(), baseURL: tt.mockServer.URL}

			defer tt.mockServer.Close()

			got, err := client.fetchLatestK3sVersion(tt.channel)
			if (err != nil) && (!tt.expectedError) {
				t.Errorf("fetchLatestK3sVersion() error = %v, wantErr %v", err, tt.errorString)
				return
			}
			if tt.expectedError {
				errorString := tt.errorString
				if tt.constructErrString {
					errorString = fmt.Sprintf(tt.errorString, tt.mockServer.URL)
				}
				if err.Error() != errorString {
					t.Errorf("expected error '%s', but got '%s'", errorString, err.Error())
				}
			}
			if got != tt.expected {
				t.Errorf("fetchLatestK3sVersion() got = %v, want %v", got, tt.expected)
			}
		})
	}
}

func mockServer(body []byte, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		writer.WriteHeader(statusCode)
		_, err := writer.Write(body)
		if err != nil {
			log.Fatalln(err)
		}
	}))
}
