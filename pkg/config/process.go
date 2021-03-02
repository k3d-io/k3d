/*
Copyright © 2020 The k3d Author(s)

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

package config

import (
	conf "github.com/rancher/k3d/v4/pkg/config/v1alpha2"
	log "github.com/sirupsen/logrus"
)

// ProcessClusterConfig applies processing to the config sanitizing it and doing
// some final modifications
func ProcessClusterConfig(clusterConfig conf.ClusterConfig) (*conf.ClusterConfig, error) {
	cluster := clusterConfig.Cluster
	if cluster.Network.Name == "host" {
		log.Infoln("Hostnetwork selected - disabling injection of docker host into the cluster, server load balancer and setting the api port to the k3s default")
		// if network is set to host, exposed api port must be the one imposed by k3s
		k3sPort := cluster.KubeAPI.Port.Port()
		log.Debugf("Host network was chosen, changing provided/random api port to k3s:%s", k3sPort)
		cluster.KubeAPI.PortMapping.Binding.HostPort = k3sPort

		// if network is host, dont inject docker host into the cluster
		clusterConfig.ClusterCreateOpts.PrepDisableHostIPInjection = true

		// if network is host, disable load balancer
		// serverlb not supported in hostnetwork mode due to port collisions with server node
		clusterConfig.ClusterCreateOpts.DisableLoadBalancer = true
	}

	return &clusterConfig, nil
}
