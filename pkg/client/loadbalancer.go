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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/go-test/deep"
	"github.com/imdario/mergo"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

var (
	ErrLBConfigHostNotFound error = errors.New("lbconfig: host not found")
	ErrLBConfigFailedTest   error = errors.New("lbconfig: failed to test")
	ErrLBConfigEntryExists  error = errors.New("lbconfig: entry exists in config")
)

// UpdateLoadbalancerConfig updates the loadbalancer config with an updated list of servers belonging to that cluster
func UpdateLoadbalancerConfig(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster) error {
	var err error
	// update cluster details to ensure that we have the latest node list
	cluster, err = ClusterGet(ctx, runtime, cluster)
	if err != nil {
		return fmt.Errorf("failed to update details for cluster '%s': %w", cluster.Name, err)
	}

	currentConfig, err := GetLoadbalancerConfig(ctx, runtime, cluster)
	if err != nil {
		return fmt.Errorf("error getting current config from loadbalancer: %w", err)
	}

	l.Log().Tracef("Current loadbalancer config:\n%+v", currentConfig)

	newLBConfig, err := LoadbalancerGenerateConfig(cluster)
	if err != nil {
		return fmt.Errorf("error generating new loadbalancer config: %w", err)
	}
	l.Log().Tracef("New loadbalancer config:\n%+v", currentConfig)

	if diff := deep.Equal(currentConfig, newLBConfig); diff != nil {
		l.Log().Debugf("Updating the loadbalancer with this diff: %+v", diff)
	}

	newLbConfigYaml, err := yaml.Marshal(&newLBConfig)
	if err != nil {
		return fmt.Errorf("error marshalling the new loadbalancer config: %w", err)
	}
	l.Log().Debugf("Writing lb config:\n%s", string(newLbConfigYaml))
	startTime := time.Now().Truncate(time.Second).UTC()
	if err := runtime.WriteToNode(ctx, newLbConfigYaml, k3d.DefaultLoadbalancerConfigPath, 0744, cluster.ServerLoadBalancer.Node); err != nil {
		return fmt.Errorf("error writing new loadbalancer config to container: %w", err)
	}

	successCtx, successCtxCancel := context.WithDeadline(ctx, time.Now().Add(5*time.Second))
	defer successCtxCancel()
	err = NodeWaitForLogMessage(successCtx, runtime, cluster.ServerLoadBalancer.Node, k3d.GetReadyLogMessage(cluster.ServerLoadBalancer.Node, k3d.IntentAny), startTime)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			failureCtx, failureCtxCancel := context.WithDeadline(ctx, time.Now().Add(5*time.Second))
			defer failureCtxCancel()
			err = NodeWaitForLogMessage(failureCtx, runtime, cluster.ServerLoadBalancer.Node, "host not found in upstream", startTime)
			if err != nil {
				l.Log().Warnf("Failed to check if the loadbalancer was configured correctly or if it broke. Please check it manually or try again: %v", err)
				return ErrLBConfigFailedTest
			} else {
				l.Log().Warnln("Failed to configure loadbalancer because one of the nodes seems to be down! Run `k3d node list` to see which one it could be.")
				return ErrLBConfigHostNotFound
			}
		} else {
			l.Log().Warnf("Failed to ensure that loadbalancer was configured correctly. Please check it manually or try again: %v", err)
			return ErrLBConfigFailedTest
		}
	}
	l.Log().Infof("Successfully configured loadbalancer %s!", cluster.ServerLoadBalancer.Node.Name)

	time.Sleep(1 * time.Second) // waiting for a second, to avoid issues with too fast lb updates which would screw up the log waits

	return nil
}

func GetLoadbalancerConfig(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster) (k3d.LoadbalancerConfig, error) {
	var cfg k3d.LoadbalancerConfig

	if cluster.ServerLoadBalancer == nil || cluster.ServerLoadBalancer.Node == nil {
		cluster.ServerLoadBalancer = &k3d.Loadbalancer{}
		for _, node := range cluster.Nodes {
			if node.Role == k3d.LoadBalancerRole {
				var err error
				cluster.ServerLoadBalancer.Node, err = NodeGet(ctx, runtime, node)
				if err != nil {
					return cfg, fmt.Errorf("failed to get loadbalancer node '%s': %w", node.Name, err)
				}
			}
		}
	}

	reader, err := runtime.ReadFromNode(ctx, k3d.DefaultLoadbalancerConfigPath, cluster.ServerLoadBalancer.Node)
	if err != nil {
		return cfg, fmt.Errorf("runtime failed to read loadbalancer config '%s' from node '%s': %w", k3d.DefaultLoadbalancerConfigPath, cluster.ServerLoadBalancer.Node.Name, err)
	}
	defer reader.Close()

	file, err := io.ReadAll(reader)
	if err != nil {
		return cfg, fmt.Errorf("failed to read loadbalancer config file: %w", err)
	}

	file = bytes.Trim(file[512:], "\x00") // trim control characters, etc.

	if err := yaml.Unmarshal(file, &cfg); err != nil {
		return cfg, fmt.Errorf("error unmarshalling loadbalancer config: %w", err)
	}

	return cfg, nil
}

func LoadbalancerGenerateConfig(cluster *k3d.Cluster) (k3d.LoadbalancerConfig, error) {
	lbConfig := k3d.LoadbalancerConfig{
		Ports:    map[string][]string{},
		Settings: k3d.LoadBalancerSettings{},
	}

	// get list of server nodes
	servers := []string{}
	for _, node := range cluster.Nodes {
		if node.Role == k3d.ServerRole {
			servers = append(servers, node.Name)
		}
	}

	// Default API Port proxied to the server nodes
	lbConfig.Ports[fmt.Sprintf("%s.tcp", k3d.DefaultAPIPort)] = servers

	// generate comma-separated list of extra ports to forward // TODO: no default targets?
	for exposedPort := range cluster.ServerLoadBalancer.Node.Ports {
		// TODO: catch duplicates here?
		lbConfig.Ports[fmt.Sprintf("%s.%s", exposedPort.Port(), exposedPort.Proto())] = servers
	}

	// some additional nginx settings
	lbConfig.Settings.WorkerConnections = k3d.DefaultLoadbalancerWorkerConnections + len(cluster.ServerLoadBalancer.Node.Ports)*len(servers)

	return lbConfig, nil
}

func LoadbalancerPrepare(ctx context.Context, runtime runtimes.Runtime, cluster *k3d.Cluster, opts *k3d.LoadbalancerCreateOpts) (*k3d.Node, error) {
	labels := map[string]string{}

	if opts != nil && opts.Labels == nil && len(opts.Labels) == 0 {
		labels = opts.Labels
	}

	if cluster.ServerLoadBalancer.Node.Ports == nil {
		cluster.ServerLoadBalancer.Node.Ports = nat.PortMap{}
	}
	cluster.ServerLoadBalancer.Node.Ports[k3d.DefaultAPIPort] = []nat.PortBinding{cluster.KubeAPI.Binding}

	if cluster.ServerLoadBalancer.Config == nil {
		cluster.ServerLoadBalancer.Config = &k3d.LoadbalancerConfig{
			Ports: map[string][]string{},
		}
	}

	if opts != nil && opts.ConfigOverrides != nil && len(opts.ConfigOverrides) > 0 {
		tmpViper := viper.New()
		for _, override := range opts.ConfigOverrides {
			kv := strings.SplitN(override, "=", 2)
			l.Log().Tracef("Overriding LB config with %s...", kv)
			tmpViper.Set(kv[0], kv[1])
		}
		lbConfigOverride := &k3d.LoadbalancerConfig{}
		if err := tmpViper.Unmarshal(lbConfigOverride); err != nil {
			return nil, fmt.Errorf("failed to unmarshal loadbalancer config override into loadbalancer config: %w", err)
		}
		if err := mergo.MergeWithOverwrite(cluster.ServerLoadBalancer.Config, lbConfigOverride); err != nil {
			return nil, fmt.Errorf("failed to override loadbalancer config: %w", err)
		}
	}

	// Create LB as a modified node with loadbalancerRole
	lbNode := &k3d.Node{
		Name:          fmt.Sprintf("%s-%s-serverlb", k3d.DefaultObjectNamePrefix, cluster.Name),
		Image:         k3d.GetLoadbalancerImage(),
		Ports:         cluster.ServerLoadBalancer.Node.Ports,
		Role:          k3d.LoadBalancerRole,
		RuntimeLabels: labels, // TODO: createLoadBalancer: add more expressive labels
		Networks:      []string{cluster.Network.Name},
		Restart:       true,
	}

	return lbNode, nil
}

func loadbalancerAddPortConfigs(loadbalancer *k3d.Loadbalancer, portmapping nat.PortMapping, targetNodes []*k3d.Node) error {
	portconfig := fmt.Sprintf("%s.%s", portmapping.Port.Port(), portmapping.Port.Proto())
	nodenames := []string{}
	for _, node := range targetNodes {
		if node.Role == k3d.LoadBalancerRole {
			return fmt.Errorf("cannot add port config referencing the loadbalancer itself (loop)")
		}
		nodenames = append(nodenames, node.Name)
	}

	// entry for that port doesn't exist yet, so we simply create it with the list of node names
	if _, ok := loadbalancer.Config.Ports[portconfig]; !ok {
		loadbalancer.Config.Ports[portconfig] = nodenames
		return nil
	}

nodenameLoop:
	for _, nodename := range nodenames {
		for _, existingNames := range loadbalancer.Config.Ports[portconfig] {
			if nodename == existingNames {
				continue nodenameLoop
			}
			loadbalancer.Config.Ports[portconfig] = append(loadbalancer.Config.Ports[portconfig], nodename)
		}
	}

	return nil
}
