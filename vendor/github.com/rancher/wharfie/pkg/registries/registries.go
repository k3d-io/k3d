package registries

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.uber.org/multierr"
	"gopkg.in/yaml.v2"
)

// registry stores information necessary to configure authentication and
// connections to remote registries, including overriding registry endpoints
type registry struct {
	DefaultKeychain authn.Keychain
	Registry        *Registry

	transports map[string]*http.Transport
}

// getPrivateRegistries loads private registry configuration from a given file
// If no file exists at the given path, default settings are returned.
// Errors such as unreadable files or unparseable content are raised.
func GetPrivateRegistries(path string) (*registry, error) {
	registry := &registry{
		DefaultKeychain: authn.DefaultKeychain,
		Registry:        &Registry{},
		transports:      map[string]*http.Transport{},
	}
	privRegistryFile, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return registry, nil
		}
		return nil, err
	}
	logrus.Infof("Using private registry config file at %s", path)
	if err := yaml.Unmarshal(privRegistryFile, registry.Registry); err != nil {
		return nil, err
	}
	return registry, nil
}

func (r *registry) Image(ref name.Reference, options ...remote.Option) (v1.Image, error) {
	endpoints, err := r.getEndpoints(ref)
	if err != nil {
		return nil, err
	}

	errs := []error{}
	for _, endpoint := range endpoints {
		epRef := ref
		if !endpoint.isDefault() {
			epRef = r.rewrite(ref)
		}
		logrus.Debugf("Trying endpoint %s", endpoint.url)
		endpointOptions := append(options, remote.WithTransport(endpoint), remote.WithAuthFromKeychain(endpoint))
		remoteImage, err := remote.Image(epRef, endpointOptions...)
		if err != nil {
			logrus.Warnf("Failed to get image from endpoint: %v", err)
			errs = append(errs, err)
			continue
		}
		return remoteImage, nil
	}
	return nil, errors.Wrap(multierr.Combine(errs...), "all endpoints failed")
}

// rewrite applies repository rewrites to the given image reference.
func (r *registry) rewrite(ref name.Reference) name.Reference {
	registry := ref.Context().RegistryStr()
	rewrites := r.getRewrites(registry)
	repository := ref.Context().RepositoryStr()

	for pattern, replace := range rewrites {
		exp, err := regexp.Compile(pattern)
		if err != nil {
			logrus.Warnf("Failed to compile rewrite `%s` for %s", pattern, registry)
			continue
		}
		if rr := exp.ReplaceAllString(repository, replace); rr != repository {
			newRepo, err := name.NewRepository(registry + "/" + rr)
			if err != nil {
				logrus.Warnf("Invalid repository rewrite %s for %s", rr, registry)
				continue
			}
			if t, ok := ref.(name.Tag); ok {
				t.Repository = newRepo
				return t
			} else if d, ok := ref.(name.Digest); ok {
				d.Repository = newRepo
				return d
			}
		}
	}

	return ref
}

// getTransport returns a transport for a given endpoint URL. For HTTP endpoints,
// the default transport is used. For HTTPS endpoints, a unique transport is created
// with the endpoint's TLSConfig (if any), and cached for all connections to this host.
func (r *registry) getTransport(endpointURL *url.URL) http.RoundTripper {
	if endpointURL.Scheme == "https" {
		// Create and cache transport if not found.
		if _, ok := r.transports[endpointURL.Host]; !ok {
			tlsConfig, err := r.getTLSConfig(endpointURL)
			if err != nil {
				logrus.Warnf("Failed to get TLS config for endpoint %v: %v", endpointURL, err)
			}

			r.transports[endpointURL.Host] = &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSClientConfig:       tlsConfig,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			}
		}
		return r.transports[endpointURL.Host]
	}
	return remote.DefaultTransport
}

// getEndpoints gets endpoint configurations for an image reference.
// The returned endpoint can be used as both a RoundTripper for requests, and a Keychain for authentication.
//
// Endpoint list generation is copied from containerd. For example, when pulling an image from gcr.io:
// * `gcr.io` is configured: endpoints for `gcr.io` + default endpoint `https://gcr.io/v2`.
// * `*` is configured, and `gcr.io` is not: endpoints for `*` + default endpoint `https://gcr.io/v2`.
// * None of above is configured: default endpoint `https://gcr.io/v2`.
func (r *registry) getEndpoints(ref name.Reference) ([]endpoint, error) {
	endpoints := []endpoint{}
	registry := ref.Context().RegistryStr()
	keys := []string{registry}
	if registry == name.DefaultRegistry {
		keys = append(keys, "docker.io")
	} else if _, _, err := net.SplitHostPort(registry); err != nil {
		keys = append(keys, registry+":443", registry+":80")
	}
	keys = append(keys, "*")

	for _, key := range keys {
		if mirror, ok := r.Registry.Mirrors[key]; ok {
			for _, endpointStr := range mirror.Endpoints {
				if endpointURL, err := normalizeEndpointAddress(endpointStr); err != nil {
					logrus.Warnf("Ignoring invalid endpoint %s for registry %s: %v", endpointStr, registry, err)
				} else {
					endpoints = append(endpoints, r.makeEndpoint(endpointURL, ref))
				}
			}
			// found a mirror for this registry, don't check any further entries
			// even if we didn't add any valid endpoints.
			break
		}
	}

	// always add the default endpoint
	defaultURL, err := normalizeEndpointAddress(registry)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to construct default endpoint for registry %s", registry)
	}
	endpoints = append(endpoints, r.makeEndpoint(defaultURL, ref))
	return endpoints, nil
}

// makeEndpoint is a utility function to create an endpoint struct for a given endpoint URL
// and registry name.
func (r *registry) makeEndpoint(endpointURL *url.URL, ref name.Reference) endpoint {
	return endpoint{
		auth:     r.getAuthenticator(endpointURL),
		keychain: r.DefaultKeychain,
		ref:      ref,
		registry: r,
		url:      endpointURL,
	}
}

// normalizeEndpointAddress normalizes the endpoint address.
// If successful, it returns the registry URL.
// If unsuccessful, an error is returned.
// Scheme and hostname logic should match containerd:
// https://github.com/containerd/containerd/blob/v1.7.13/remotes/docker/config/hosts.go#L99-L131
func normalizeEndpointAddress(endpoint string) (*url.URL, error) {
	// Ensure that the endpoint address has a scheme so that the URL is parsed properly
	if !strings.Contains(endpoint, "://") {
		endpoint = "//" + endpoint
	}
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	if endpointURL.Host == "" {
		return nil, fmt.Errorf("invalid URL without host: %s", endpoint)
	}
	if endpointURL.Scheme == "" {
		// localhost on odd ports defaults to http
		port := endpointURL.Port()
		if isLocalhost(endpointURL.Host) && port != "" && port != "443" {
			endpointURL.Scheme = "http"
		} else {
			endpointURL.Scheme = "https"
		}
	}
	switch endpointURL.Path {
	case "", "/", "/v2":
		endpointURL.Path = "/v2"
	default:
		endpointURL.Path = path.Clean(endpointURL.Path)
	}
	return endpointURL, nil
}

// getAuthenticatorForHost returns an Authenticator for an endpoint URL. If no
// configuration is present, Anonymous authentication is used.
func (r *registry) getAuthenticator(endpointURL *url.URL) authn.Authenticator {
	registry := endpointURL.Host
	keys := []string{registry}
	if registry == name.DefaultRegistry {
		keys = append(keys, "docker.io")
	}
	keys = append(keys, "*")

	for _, key := range keys {
		if config, ok := r.Registry.Configs[key]; ok {
			if config.Auth != nil {
				return authn.FromConfig(authn.AuthConfig{
					Username:      config.Auth.Username,
					Password:      config.Auth.Password,
					Auth:          config.Auth.Auth,
					IdentityToken: config.Auth.IdentityToken,
				})
			}
			// found a config for this registry, don't check any further entries
			// even if we didn't add any valid auth.
			break
		}
	}
	return authn.Anonymous
}

// getTLSConfig returns TLS configuration for an endpoint URL. This is cribbed from
// https://github.com/containerd/cri/blob/release/1.4/pkg/server/image_pull.go#L274
func (r *registry) getTLSConfig(endpointURL *url.URL) (*tls.Config, error) {
	tlsConfig := &tls.Config{}
	registry := endpointURL.Host
	keys := []string{registry}
	if registry == name.DefaultRegistry {
		keys = append(keys, "docker.io")
	}
	keys = append(keys, "*")

	for _, key := range keys {
		if config, ok := r.Registry.Configs[key]; ok {
			if config.TLS != nil {
				if config.TLS.CertFile != "" && config.TLS.KeyFile == "" {
					return nil, errors.Errorf("cert file %q was specified, but no corresponding key file was specified", config.TLS.CertFile)
				}
				if config.TLS.CertFile == "" && config.TLS.KeyFile != "" {
					return nil, errors.Errorf("key file %q was specified, but no corresponding cert file was specified", config.TLS.KeyFile)
				}
				if config.TLS.CertFile != "" && config.TLS.KeyFile != "" {
					cert, err := tls.LoadX509KeyPair(config.TLS.CertFile, config.TLS.KeyFile)
					if err != nil {
						return nil, errors.Wrap(err, "failed to load cert file")
					}
					if len(cert.Certificate) != 0 {
						tlsConfig.Certificates = []tls.Certificate{cert}
					}
					tlsConfig.BuildNameToCertificate() // nolint:staticcheck
				}

				if config.TLS.CAFile != "" {
					caCertPool, err := x509.SystemCertPool()
					if err != nil {
						return nil, errors.Wrap(err, "failed to get system cert pool")
					}
					caCert, err := ioutil.ReadFile(config.TLS.CAFile)
					if err != nil {
						return nil, errors.Wrap(err, "failed to load CA file")
					}
					caCertPool.AppendCertsFromPEM(caCert)
					tlsConfig.RootCAs = caCertPool
				}

				tlsConfig.InsecureSkipVerify = config.TLS.InsecureSkipVerify
			}
			// found a config for this registry, don't check any further entries
			// even if we didn't add any valid tls config.
			break
		}
	}

	return tlsConfig, nil
}

// getRewritesForHost gets the map of rewrite patterns for a given registry.
func (r *registry) getRewrites(registry string) map[string]string {
	keys := []string{registry}
	if registry == name.DefaultRegistry {
		keys = append(keys, "docker.io")
	}
	keys = append(keys, "*")

	for _, key := range keys {
		if mirror, ok := r.Registry.Mirrors[key]; ok {
			if len(mirror.Rewrites) > 0 {
				return mirror.Rewrites
			}
			// found a mirror for this registry, don't check any further entries
			// even if we didn't add any rewrites.
			break
		}
	}

	return nil
}

func isLocalhost(host string) bool {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	if host == "localhost" {
		return true
	}

	ip := net.ParseIP(host)
	return ip.IsLoopback()
}
