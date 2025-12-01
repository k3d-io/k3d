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

package client

import (
	"testing"

	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

func TestAPIHostReplacement(t *testing.T) {
	tests := []struct {
		name            string
		inputHost       string
		expectedHost    string
		shouldBeChanged bool
	}{
		{
			name:            "0.0.0.0 should be replaced with 127.0.0.1",
			inputHost:       "0.0.0.0",
			expectedHost:    "127.0.0.1",
			shouldBeChanged: true,
		},
		{
			name:            "127.0.0.1 should remain unchanged",
			inputHost:       "127.0.0.1",
			expectedHost:    "127.0.0.1",
			shouldBeChanged: false,
		},
		{
			name:            "localhost should remain unchanged",
			inputHost:       "localhost",
			expectedHost:    "localhost",
			shouldBeChanged: false,
		},
		{
			name:            "custom IP should remain unchanged",
			inputHost:       "192.168.1.100",
			expectedHost:    "192.168.1.100",
			shouldBeChanged: false,
		},
		{
			name:            "hostname should remain unchanged",
			inputHost:       "my-server.example.com",
			expectedHost:    "my-server.example.com",
			shouldBeChanged: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiHost := tt.inputHost
			// This is the logic from KubeconfigGet that replaces 0.0.0.0 with 127.0.0.1
			if apiHost == k3d.DefaultAPIHost {
				apiHost = "127.0.0.1"
			}

			if apiHost != tt.expectedHost {
				t.Errorf("Expected API host to be '%s', but got '%s'", tt.expectedHost, apiHost)
			}
		})
	}
}
