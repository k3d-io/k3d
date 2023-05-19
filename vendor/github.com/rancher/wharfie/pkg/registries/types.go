package registries

// Mirror contains the config related to the registry mirror
type Mirror struct {
	// Endpoints are endpoints for a namespace. CRI plugin will try the endpoints
	// one by one until a working one is found. The endpoint must be a valid url
	// with host specified.
	// The scheme, host and path from the endpoint URL will be used.
	Endpoints []string `toml:"endpoint" yaml:"endpoint" json:"endpoint"`

	// Rewrites are repository rewrite rules for a namespace. When fetching image resources
	// from an endpoint and a key matches the repository via regular expression matching
	// it will be replaced with the corresponding value from the map in the resource request.
	Rewrites map[string]string `toml:"rewrite" yaml:"rewrite" json:"rewrite"`
}

// AuthConfig contains the config related to authentication to a specific registry
type AuthConfig struct {
	// Username is the username to login the registry.
	Username string `toml:"username" yaml:"username" json:"username"`
	// Password is the password to login the registry.
	Password string `toml:"password" yaml:"password" json:"password"`
	// Auth is a base64 encoded string from the concatenation of the username,
	// a colon, and the password.
	Auth string `toml:"auth" yaml:"auth" json:"auth"`
	// IdentityToken is used to authenticate the user and get
	// an access token for the registry.
	IdentityToken string `toml:"identitytoken" yaml:"identity_token" json:"identitytoken"`
}

// TLSConfig contains the CA/Cert/Key used for a registry
type TLSConfig struct {
	CAFile             string `toml:"ca_file" yaml:"ca_file" json:"ca_file"`
	CertFile           string `toml:"cert_file" yaml:"cert_file" json:"cert_file"`
	KeyFile            string `toml:"key_file" yaml:"key_file" json:"key_file"`
	InsecureSkipVerify bool   `toml:"insecure_skip_verify" yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
}

// Registry is registry settings including mirrors, TLS, and credentials
type Registry struct {
	// Mirrors are namespace to mirror mapping for all namespaces.
	Mirrors map[string]Mirror `toml:"mirrors" yaml:"mirrors" json:"mirrors"`
	// Configs are configs for each registry.
	// The key is the FDQN or IP of the registry.
	Configs map[string]RegistryConfig `toml:"configs" yaml:"configs" json:"configs"`

	// Auths are registry endpoint to auth config mapping. The registry endpoint must
	// be a valid url with host specified.
	// DEPRECATED: Use Configs instead. Remove in containerd 1.4.
	Auths map[string]AuthConfig `toml:"auths" yaml:"auths" json:"auths"`
}

// RegistryConfig contains configuration used to communicate with the registry.
type RegistryConfig struct {
	// Auth contains information to authenticate to the registry.
	Auth *AuthConfig `toml:"auth" yaml:"auth" json:"auth"`
	// TLS is a pair of CA/Cert/Key which then are used when creating the transport
	// that communicates with the registry.
	TLS *TLSConfig `toml:"tls" yaml:"tls" json:"tls"`
}
