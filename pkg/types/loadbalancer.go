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
package types

/* DESCRIPTION
 * The Loadbalancer is a customized NGINX container running side-by-side with the cluster, NOT INSIDE IT.
 * It is used to do plain proxying of tcp/udp ports to the k3d node containers.
 * One advantage of this approach is, that we can add new ports while the cluster is still running by re-creating
 * the loadbalancer and adding the new port config in the NGINX config. As the loadbalancer doesn't hold any state
 * (apart from the config file), it can easily be re-created in just a few seconds.
 */

/*
 * Loadbalancer Definition
 */

type Loadbalancer struct {
	*Node  `mapstructure:",squash"` // the underlying node
	Config *LoadbalancerConfig      `mapstructure:"config" json:"config"` // its configuration
}

func NewLoadbalancer() *Loadbalancer {
	return &Loadbalancer{
		Node: &Node{
			Role:  LoadBalancerRole,
			Image: GetLoadbalancerImage(),
		},
		Config: &LoadbalancerConfig{
			Ports: map[string][]string{},
			Settings: LoadBalancerSettings{
				WorkerConnections: DefaultLoadbalancerWorkerConnections,
			},
		},
	}
}

/*
 * Loadbalancer Configuration
 */

/* LoadbalancerConfig defines the coarse file structure to configure the k3d-proxy
 * Example:
 * ports:
 * 	1234.tcp:
 * 		- k3d-k3s-default-server-0
 * 		- k3d-k3s-default-server-1
 * 	4321.udp:
 * 		- k3d-k3s-default-agent-0
 * 		- k3d-k3s-default-agent-1
 */
type LoadbalancerConfig struct {
	Ports    map[string][]string  `json:"ports"`
	Settings LoadBalancerSettings `json:"settings"`
}

type LoadBalancerSettings struct {
	WorkerConnections   int `json:"workerConnections"`
	DefaultProxyTimeout int `json:"defaultProxyTimeout,omitempty"`
}

const (
	DefaultLoadbalancerConfigPath        = "/etc/confd/values.yaml"
	DefaultLoadbalancerWorkerConnections = 1024
)

type LoadbalancerCreateOpts struct {
	Labels          map[string]string
	ConfigOverrides []string
}

/*
 * Helper Functions
 */

// HasLoadBalancer returns true if cluster has a loadbalancer node
func (c *Cluster) HasLoadBalancer() bool {
	for _, node := range c.Nodes {
		if node.Role == LoadBalancerRole {
			return true
		}
	}
	return false
}
