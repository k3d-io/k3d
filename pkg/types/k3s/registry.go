/*
Copyright © 2020-2022 The k3d Author(s)

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
package k3s

/*
 * Copied from https://github.com/k3s-io/k3s/blob/cf8c101b705c7af20e2ed11df43beb4951e6d9dc/pkg/agent/templates/registry.go
 * .. to avoid pulling in k3s as a dependency
 */

// Mirror contains the config related to the registry mirror
type Mirror struct {
	// Endpoints are endpoints for a namespace. CRI plugin will try the endpoints
	// one by one until a working one is found. The endpoint must be a valid url
	// with host specified.
	// The scheme, host and path from the endpoint URL will be used.
	Endpoints []string `toml:"endpoint" json:"endpoint"`
}

// AuthConfig contains the config related to authentication to a specific registry
type AuthConfig struct {
	// Username is the username to login the registry.
	Username string `toml:"username" json:"username"`
	// Password is the password to login the registry.
	Password string `toml:"password" json:"password"`
	// Auth is a base64 encoded string from the concatenation of the username,
	// a colon, and the password.
	Auth string `toml:"auth" json:"auth"`
	// IdentityToken is used to authenticate the user and get
	// an access token for the registry.
	IdentityToken string `toml:"identitytoken" json:"identity_token"`
}

// TLSConfig contains the CA/Cert/Key used for a registry
type TLSConfig struct {
	CAFile             string `toml:"ca_file" json:"ca_file"`
	CertFile           string `toml:"cert_file" json:"cert_file"`
	KeyFile            string `toml:"key_file" json:"key_file"`
	InsecureSkipVerify bool   `toml:"insecure_skip_verify" json:"insecure_skip_verify"`
}

// Registry is registry settings configured
type Registry struct {
	// Mirrors are namespace to mirror mapping for all namespaces.
	Mirrors map[string]Mirror `toml:"mirrors" json:"mirrors"`
	// Configs are configs for each registry.
	// The key is the FDQN or IP of the registry.
	Configs map[string]RegistryConfig `toml:"configs" json:"configs"`

	// Auths are registry endpoint to auth config mapping. The registry endpoint must
	// be a valid url with host specified.
	// DEPRECATED: Use Configs instead. Remove in containerd 1.4.
	Auths map[string]AuthConfig `toml:"auths" json:"auths"`
}

// RegistryConfig contains configuration used to communicate with the registry.
type RegistryConfig struct {
	// Auth contains information to authenticate to the registry.
	Auth *AuthConfig `toml:"auth" json:"auth"`
	// TLS is a pair of CA/Cert/Key which then are used when creating the transport
	// that communicates with the registry.
	TLS *TLSConfig `toml:"tls" json:"tls"`
}
