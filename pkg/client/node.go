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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	dockerunits "github.com/docker/go-units"
	"github.com/imdario/mergo"
	copystruct "github.com/mitchellh/copystructure"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/yaml"

	"github.com/k3d-io/k3d/v5/pkg/actions"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	"github.com/k3d-io/k3d/v5/pkg/runtimes/docker"
	runtimeErrors "github.com/k3d-io/k3d/v5/pkg/runtimes/errors"
	runtimeTypes "github.com/k3d-io/k3d/v5/pkg/runtimes/types"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/pkg/types/fixes"
	"github.com/k3d-io/k3d/v5/pkg/types/k3s"
	"github.com/k3d-io/k3d/v5/pkg/util"
	"github.com/k3d-io/k3d/v5/version"
)

// NodeAddToCluster adds a node to an existing cluster
func NodeAddToCluster(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, cluster *k3d.Cluster, createNodeOpts k3d.NodeCreateOpts) error {
	if createNodeOpts.EnvironmentInfo == nil {
		targetClusterName := cluster.Name
		cluster, err := ClusterGet(ctx, runtime, cluster)
		if err != nil {
			return fmt.Errorf("Failed to find specified cluster '%s': %w", targetClusterName, err)
		}

		envInfo, err := GatherEnvironmentInfo(ctx, runtime, cluster)
		if err != nil {
			return fmt.Errorf("error gathering cluster environment info required to properly create the node: %w", err)
		}

		createNodeOpts.EnvironmentInfo = envInfo
	}

	// networks: ensure that cluster network is on index 0
	networks := []string{cluster.Network.Name}
	if node.Networks != nil {
		networks = append(networks, node.Networks...)
	}
	node.Networks = networks

	// skeleton
	if node.RuntimeLabels == nil {
		node.RuntimeLabels = map[string]string{
			k3d.LabelRole: string(node.Role),
		}
	}
	node.Env = []string{}

	// copy labels and env vars from a similar node in the selected cluster
	var srcNode *k3d.Node
	for _, existingNode := range cluster.Nodes {
		if existingNode.Role == node.Role {
			srcNode = existingNode
			break
		}
	}
	// if we didn't find a node with the same role in the cluster, just choose any other node
	if srcNode == nil {
		l.Log().Debugf("Didn't find node with role '%s' in cluster '%s'. Choosing any other node (and using defaults)...", node.Role, cluster.Name)
		node.Cmd = k3d.DefaultRoleCmds[node.Role]
		for _, existingNode := range cluster.Nodes {
			if existingNode.Role == k3d.AgentRole || existingNode.Role == k3d.ServerRole { // only K3s nodes
				srcNode = existingNode
				break
			}
		}
	}

	// get node details
	srcNode, err := NodeGet(ctx, runtime, srcNode)
	if err != nil {
		return err
	}

	/*
	 * Sanitize Source Node
	 * -> remove fields that are not safe to copy as they break something down the stream
	 */

	// sanitize fields that mismatch between roles
	if srcNode.Role != node.Role {
		l.Log().Debugf("Dropping some fields from source node because it's not of the same role (%s != %s)...", srcNode.Role, node.Role)
		srcNode.Memory = "" // memory settings are scoped per role (--servers-memory/--agents-memory)
	}

	// TODO: I guess proper deduplication can be handled in a cleaner/better way or at the infofaker level at some point
	for _, forbiddenMount := range util.DoNotCopyVolumeSuffices {
		for i, mount := range srcNode.Volumes {
			if strings.Contains(mount, forbiddenMount) {
				l.Log().Tracef("Dropping volume mount %s from source node to avoid issues...", mount)
				srcNode.Volumes = util.RemoveElementFromStringSlice(srcNode.Volumes, i)
			}
		}
	}

	// drop port mappings as we  cannot use the same port mapping for a two nodes (port collisions)
	srcNode.Ports = nat.PortMap{}

	// we cannot have two servers as init servers
	if node.Role == k3d.ServerRole {
		for _, forbiddenCmd := range k3d.DoNotCopyServerFlags {
			for i, cmd := range srcNode.Cmd {
				// cut out the '--cluster-init' flag as this should only be done by the initializing server node
				if cmd == forbiddenCmd {
					l.Log().Tracef("Dropping '%s' from source node's cmd", forbiddenCmd)
					srcNode.Cmd = append(srcNode.Cmd[:i], srcNode.Cmd[i+1:]...)
				}
			}
			for i, arg := range node.Args {
				// cut out the '--cluster-init' flag as this should only be done by the initializing server node
				if arg == forbiddenCmd {
					l.Log().Tracef("Dropping '%s' from source node's args", forbiddenCmd)
					srcNode.Args = append(srcNode.Args[:i], srcNode.Args[i+1:]...)
				}
			}
		}
	}

	l.Log().Debugf("Adding node %s to cluster %s based on existing (sanitized) node %s", node.Name, cluster.Name, srcNode.Name)
	l.Log().Tracef("Sanitized Source Node: %+v\nNew Node: %+v", srcNode, node)

	// fetch registry config
	registryConfigBytes := []byte{}
	registryConfigReader, err := runtime.ReadFromNode(ctx, k3d.DefaultRegistriesFilePath, srcNode)
	if err != nil {
		if !errors.Is(err, runtimeErrors.ErrRuntimeFileNotFound) {
			l.Log().Warnf("Failed to read registry config from node %s: %+v", node.Name, err)
		}
	} else {
		defer registryConfigReader.Close()

		var err error
		registryConfigBytes, err = io.ReadAll(registryConfigReader)
		if err != nil {
			l.Log().Warnf("Failed to read registry config from node %s: %+v", node.Name, err)
		}
		registryConfigReader.Close()
		registryConfigBytes = bytes.Trim(registryConfigBytes[512:], "\x00") // trim control characters, etc.
	}

	// merge node config of new node into existing node config
	if err := mergo.MergeWithOverwrite(srcNode, *node); err != nil {
		return fmt.Errorf("failed to merge new node config into existing node config: %w", err)
	}

	node = srcNode

	l.Log().Tracef("Resulting node %+v", node)

	k3sURLEnvFound := false
	k3sURLEnvIndex := -1
	k3sTokenEnvFoundIndex := -1
	for index, envVar := range node.Env {
		if strings.HasPrefix(envVar, k3s.EnvClusterConnectURL) {
			k3sURLEnvFound = true
			k3sURLEnvIndex = index
		}
		if strings.HasPrefix(envVar, k3s.EnvClusterToken) {
			k3sTokenEnvFoundIndex = index
		}
	}

	if !k3sURLEnvFound || !checkK3SURLIsActive(cluster.Nodes, node.Env[k3sURLEnvIndex]) {
		// Use LB if available as registration url for nodes
		// otherwise fallback to server node
		var registrationNode *k3d.Node
		serverLBNodes := util.FilterNodesByRole(cluster.Nodes, k3d.LoadBalancerRole)
		if len(serverLBNodes) > 0 {
			registrationNode = serverLBNodes[0]
		} else {
			for _, existingNode := range cluster.Nodes {
				if existingNode.Role == k3d.ServerRole {
					registrationNode = existingNode
					break
				}
			}
		}

		if !k3sURLEnvFound {
			node.Env = append(node.Env, fmt.Sprintf("%s=https://%s:%s", k3s.EnvClusterConnectURL, registrationNode.Name, k3d.DefaultAPIPort))
		} else {
			if !checkK3SURLIsActive(cluster.Nodes, node.Env[k3sURLEnvIndex]) {
				node.Env[k3sURLEnvIndex] = fmt.Sprintf("%s=https://%s:%s", k3s.EnvClusterConnectURL, registrationNode.Name, k3d.DefaultAPIPort)
			}
		}
	}

	if k3sTokenEnvFoundIndex != -1 && createNodeOpts.ClusterToken != "" {
		l.Log().Debugln("Overriding copied cluster token with value from nodeCreateOpts...")
		node.Env[k3sTokenEnvFoundIndex] = fmt.Sprintf("%s=%s", k3s.EnvClusterToken, createNodeOpts.ClusterToken)
		node.RuntimeLabels[k3d.LabelClusterToken] = createNodeOpts.ClusterToken
	}

	/*
	 * Add Node Hook Actions (Lifecylce Hooks)
	 */

	// Write registry config file if there is any
	if len(registryConfigBytes) != 0 {
		if createNodeOpts.NodeHooks == nil {
			createNodeOpts.NodeHooks = []k3d.NodeHook{}
		}
		createNodeOpts.NodeHooks = append(createNodeOpts.NodeHooks,
			k3d.NodeHook{
				Stage: k3d.LifecycleStagePreStart,
				Action: actions.WriteFileAction{
					Runtime:     runtime,
					Content:     registryConfigBytes,
					Dest:        k3d.DefaultRegistriesFilePath,
					Mode:        0644,
					Description: "Write Registry Configuration",
				},
			},
		)
	}

	if cluster.Network.Name != "host" {
		// add host.k3d.internal to /etc/hosts
		createNodeOpts.NodeHooks = append(createNodeOpts.NodeHooks,
			k3d.NodeHook{
				Stage: k3d.LifecycleStagePostStart,
				Action: actions.ExecAction{
					Runtime: runtime,
					Command: []string{
						"sh", "-c",
						fmt.Sprintf("echo '%s %s' >> /etc/hosts", createNodeOpts.EnvironmentInfo.HostGateway.String(), k3d.DefaultK3dInternalHostRecord),
					},
					Retries:     0,
					Description: fmt.Sprintf("Inject /etc/hosts record for %s", k3d.DefaultK3dInternalHostRecord),
				},
			},
		)
	}

	// clear status fields
	node.State.Running = false
	node.State.Status = ""

	if err := NodeRun(ctx, runtime, node, createNodeOpts); err != nil {
		return fmt.Errorf("failed to run node '%s': %w", node.Name, err)
	}

	// if it's a server node, then update the loadbalancer configuration
	if node.Role == k3d.ServerRole {
		l.Log().Infoln("Updating loadbalancer config to include new server node(s)")
		if err := UpdateLoadbalancerConfig(ctx, runtime, cluster); err != nil {
			if !errors.Is(err, ErrLBConfigHostNotFound) {
				return fmt.Errorf("error updating loadbalancer: %w", err)
			}
		}
	}

	// inject host.k3d.internal entry

	return nil
}

func NodeAddToClusterRemote(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, clusterRef string, createNodeOpts k3d.NodeCreateOpts) error {
	// runtime labels
	if node.RuntimeLabels == nil {
		node.RuntimeLabels = map[string]string{}
	}

	node.FillRuntimeLabels()

	node.RuntimeLabels[k3d.LabelClusterName] = clusterRef
	node.RuntimeLabels[k3d.LabelClusterURL] = clusterRef
	node.RuntimeLabels[k3d.LabelClusterExternal] = "true"
	node.RuntimeLabels[k3d.LabelClusterToken] = createNodeOpts.ClusterToken

	if node.Image == "" { // we don't set a default, because on local clusters the value is copied from existing nodes
		node.Image = fmt.Sprintf("%s:%s", k3d.DefaultK3sImageRepo, version.K3sVersion)
	}

	if node.Env == nil {
		node.Env = []string{}
	}

	node.Env = append(node.Env, fmt.Sprintf("%s=%s", k3s.EnvClusterConnectURL, clusterRef))
	node.Env = append(node.Env, fmt.Sprintf("%s=%s", k3s.EnvClusterToken, createNodeOpts.ClusterToken))

	if err := NodeRun(ctx, runtime, node, createNodeOpts); err != nil {
		return fmt.Errorf("failed to run node '%s': %w", node.Name, err)
	}

	return nil
}

// NodeAddToClusterMulti adds multiple nodes to a chosen cluster
func NodeAddToClusterMulti(ctx context.Context, runtime runtimes.Runtime, nodes []*k3d.Node, cluster *k3d.Cluster, createNodeOpts k3d.NodeCreateOpts) error {
	if createNodeOpts.Timeout > 0*time.Second {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, createNodeOpts.Timeout)
		defer cancel()
	}

	targetClusterName := cluster.Name
	cluster, err := ClusterGet(ctx, runtime, cluster)
	if err != nil {
		return fmt.Errorf("Failed to find specified cluster '%s': %w", targetClusterName, err)
	}

	createNodeOpts.EnvironmentInfo, err = GatherEnvironmentInfo(ctx, runtime, cluster)
	if err != nil {
		return fmt.Errorf("error gathering cluster environment info required to properly create the node: %w", err)
	}

	nodeWaitGroup, ctx := errgroup.WithContext(ctx)
	for _, node := range nodes {
		currentNode := node
		nodeWaitGroup.Go(func() error {
			return NodeAddToCluster(ctx, runtime, currentNode, cluster, createNodeOpts)
		})
	}
	if err := nodeWaitGroup.Wait(); err != nil {
		return fmt.Errorf("failed to add one or more nodes: %w", err)
	}

	return nil
}

func NodeAddToClusterMultiRemote(ctx context.Context, runtime runtimes.Runtime, nodes []*k3d.Node, clusterRef string, createNodeOpts k3d.NodeCreateOpts) error {
	if createNodeOpts.Timeout > 0*time.Second {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, createNodeOpts.Timeout)
		defer cancel()
	}

	nodeWaitGroup, ctx := errgroup.WithContext(ctx)
	for _, node := range nodes {
		currentNode := node
		nodeWaitGroup.Go(func() error {
			return NodeAddToClusterRemote(ctx, runtime, currentNode, clusterRef, createNodeOpts)
		})
	}
	if err := nodeWaitGroup.Wait(); err != nil {
		return fmt.Errorf("failed to add one or more nodes: %w", err)
	}

	return nil
}

// NodeCreateMulti creates a list of nodes
func NodeCreateMulti(ctx context.Context, runtime runtimes.Runtime, nodes []*k3d.Node, createNodeOpts k3d.NodeCreateOpts) error { // TODO: pass `--atomic` flag, so we stop and return an error if any node creation fails?
	if createNodeOpts.Timeout > 0*time.Second {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, createNodeOpts.Timeout)
		defer cancel()
	}

	nodeWaitGroup, ctx := errgroup.WithContext(ctx)
	for _, node := range nodes {
		if err := NodeCreate(ctx, runtime, node, k3d.NodeCreateOpts{}); err != nil {
			l.Log().Error(err)
		}
		if createNodeOpts.Wait {
			currentNode := node
			nodeWaitGroup.Go(func() error {
				l.Log().Debugf("Starting to wait for node '%s'", currentNode.Name)
				readyLogMessage := k3d.GetReadyLogMessage(currentNode, k3d.IntentNodeCreate)
				if readyLogMessage != "" {
					return NodeWaitForLogMessage(ctx, runtime, currentNode, readyLogMessage, time.Time{})
				}
				l.Log().Warnf("NodeCreateMulti: Set to wait for node %s to get ready, but there's no target log message defined", currentNode.Name)
				return nil
			})
		}
	}

	if err := nodeWaitGroup.Wait(); err != nil {
		return fmt.Errorf("failed to create nodes: %w", err)
	}

	return nil
}

// NodeRun creates and starts a node
func NodeRun(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, nodeCreateOpts k3d.NodeCreateOpts) error {
	if err := NodeCreate(ctx, runtime, node, nodeCreateOpts); err != nil {
		return fmt.Errorf("failed to create node '%s': %w", node.Name, err)
	}

	if err := NodeStart(ctx, runtime, node, &k3d.NodeStartOpts{
		Wait:            nodeCreateOpts.Wait,
		Timeout:         nodeCreateOpts.Timeout,
		NodeHooks:       nodeCreateOpts.NodeHooks,
		EnvironmentInfo: nodeCreateOpts.EnvironmentInfo,
		Intent:          k3d.IntentNodeCreate,
	}); err != nil {
		return fmt.Errorf("failed to start node '%s': %w", node.Name, err)
	}

	return nil
}

// NodeStart starts an existing node
func NodeStart(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, nodeStartOpts *k3d.NodeStartOpts) error {
	// return early, if the node is already running
	if node.State.Running {
		l.Log().Infof("Node %s is already running", node.Name)
		return nil
	}

	if err := enableFixes(ctx, runtime, node, nodeStartOpts); err != nil {
		return fmt.Errorf("failed to enable k3d fixes: %w", err)
	}

	startTime := time.Now()
	l.Log().Debugf("Node %s Start Time: %+v", node.Name, startTime)

	// execute lifecycle hook actions
	for _, hook := range nodeStartOpts.NodeHooks {
		if hook.Stage == k3d.LifecycleStagePreStart {
			l.Log().Tracef("Node %s: Executing preStartAction '%s': %s", node.Name, hook.Action.Name(), hook.Action.Info())
			if err := hook.Action.Run(ctx, node); err != nil {
				l.Log().Errorf("Node %s: Failed executing preStartAction '%s': %s: %+v", node.Name, hook.Action.Name(), hook.Action.Info(), err) // TODO: should we error out on failed preStartAction?
			}
		}
	}

	// start the node
	l.Log().Tracef("Starting node '%s'", node.Name)

	if err := runtime.StartNode(ctx, node); err != nil {
		return fmt.Errorf("runtime failed to start node '%s': %w", node.Name, err)
	}

	if node.State.Started != "" {
		ts, err := time.Parse("2006-01-02T15:04:05.999999999Z", node.State.Started)
		if err != nil {
			l.Log().Debugf("Failed to parse '%s.State.Started' timestamp '%s', falling back to calculated time", node.Name, node.State.Started)
		}
		startTime = ts.Truncate(time.Second)
		l.Log().Debugf("Truncated %s to %s", ts, startTime)
	}

	if nodeStartOpts.Wait {
		if nodeStartOpts.ReadyLogMessage == "" {
			nodeStartOpts.ReadyLogMessage = k3d.GetReadyLogMessage(node, nodeStartOpts.Intent)
		}
		if nodeStartOpts.ReadyLogMessage != "" {
			l.Log().Debugf("Waiting for node %s to get ready (Log: '%s')", node.Name, nodeStartOpts.ReadyLogMessage)
			if err := NodeWaitForLogMessage(ctx, runtime, node, nodeStartOpts.ReadyLogMessage, startTime); err != nil {
				return fmt.Errorf("Node %s failed to get ready: %+v", node.Name, err)
			}
		} else {
			l.Log().Warnf("NodeStart: Set to wait for node %s to be ready, but there's no target log message defined", node.Name)
		}
	}

	// execute lifecycle hook actions
	for _, hook := range nodeStartOpts.NodeHooks {
		if hook.Stage == k3d.LifecycleStagePostStart {
			l.Log().Tracef("Node %s: Executing postStartAction: %s", node.Name, hook.Action.Info())
			if err := hook.Action.Run(ctx, node); err != nil {
				return fmt.Errorf("Node %s: Failed executing postStartAction: %s: %+v", node.Name, hook.Action.Info(), err)
			}
		}
	}

	return nil
}

func enableFixes(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, nodeStartOpts *k3d.NodeStartOpts) error {
	if node.Role == k3d.ServerRole || node.Role == k3d.AgentRole {
		enabledFixes, anyEnabled := fixes.GetFixes(runtime)

		// early exit if we don't need any fix
		if !anyEnabled {
			l.Log().Debugln("No fix enabled.")
			return nil
		}

		// ensure nodehook list
		if nodeStartOpts.NodeHooks == nil {
			nodeStartOpts.NodeHooks = []k3d.NodeHook{}
		}

		// write umbrella entrypoint
		nodeStartOpts.NodeHooks = append(nodeStartOpts.NodeHooks, k3d.NodeHook{
			Stage: k3d.LifecycleStagePreStart,
			Action: actions.WriteFileAction{
				Runtime:     runtime,
				Content:     fixes.K3DEntrypoint,
				Dest:        "/bin/k3d-entrypoint.sh",
				Mode:        0744,
				Description: "Write custom k3d entrypoint script (that powers the magic fixes)",
			},
		})

		// DNS Fix
		if enabledFixes[fixes.EnvFixDNS] {
			l.Log().Debugln(">>> enabling dns magic")

			for _, v := range node.Volumes {
				if strings.Contains(v, "/etc/resolv.conf") {
					return fmt.Errorf("[Node %s] Cannot activate DNS fix (K3D_FIX_DNS) when there's a file mounted at /etc/resolv.conf!", node.Name)
				}
			}

			if nodeStartOpts.EnvironmentInfo == nil || !nodeStartOpts.EnvironmentInfo.HostGateway.IsValid() {
				return fmt.Errorf("Cannot enable DNS fix, as Host Gateway IP is missing!")
			}

			data := []byte(strings.ReplaceAll(string(fixes.DNSMagicEntrypoint), "GATEWAY_IP", nodeStartOpts.EnvironmentInfo.HostGateway.String()))

			nodeStartOpts.NodeHooks = append(nodeStartOpts.NodeHooks, k3d.NodeHook{
				Stage: k3d.LifecycleStagePreStart,
				Action: actions.WriteFileAction{
					Runtime:     runtime,
					Content:     data,
					Dest:        "/bin/k3d-entrypoint-dns.sh",
					Mode:        0744,
					Description: "Write entrypoint script for DNS fix",
				},
			})
		}

		// CGroupsV2Fix
		if enabledFixes[fixes.EnvFixCgroupV2] {
			l.Log().Debugf(">>> enabling cgroupsv2 magic")

			if nodeStartOpts.NodeHooks == nil {
				nodeStartOpts.NodeHooks = []k3d.NodeHook{}
			}

			nodeStartOpts.NodeHooks = append(nodeStartOpts.NodeHooks, k3d.NodeHook{
				Stage: k3d.LifecycleStagePreStart,
				Action: actions.WriteFileAction{
					Runtime:     runtime,
					Content:     fixes.CgroupV2Entrypoint,
					Dest:        "/bin/k3d-entrypoint-cgroupv2.sh",
					Mode:        0744,
					Description: "Write entrypoint script for CGroupV2 fix",
				},
			})
		}

		if enabledFixes[fixes.EnvFixMounts] {
			l.Log().Debugf(">>> enabling mounts magic")

			if nodeStartOpts.NodeHooks == nil {
				nodeStartOpts.NodeHooks = []k3d.NodeHook{}
			}

			nodeStartOpts.NodeHooks = append(nodeStartOpts.NodeHooks, k3d.NodeHook{
				Stage: k3d.LifecycleStagePreStart,
				Action: actions.WriteFileAction{
					Runtime:     runtime,
					Content:     fixes.MountsEntrypoint,
					Dest:        "/bin/k3d-entrypoint-mounts.sh",
					Mode:        0744,
					Description: "Write entrypoint script for mounts fix",
				},
			})
		}
	}
	return nil
}

// NodeCreate creates a new containerized k3s node
func NodeCreate(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, createNodeOpts k3d.NodeCreateOpts) error {
	l.Log().Tracef("Creating node from spec\n%+v", node)

	/*
	 * CONFIGURATION
	 */

	/* global node configuration (applies for any node role) */

	// ### Labels ###
	node.FillRuntimeLabels()

	for k, v := range node.K3sNodeLabels {
		node.Args = append(node.Args, "--node-label", fmt.Sprintf("%s=%s", k, v))
	}

	// ### Environment ###
	node.Env = append(node.Env, k3d.DefaultNodeEnv...) // append default node env vars

	// specify options depending on node role
	if node.Role == k3d.AgentRole { // TODO: check here AND in CLI or only here?
		if err := patchAgentSpec(node); err != nil {
			return fmt.Errorf("failed to patch agent spec on node %s: %w", node.Name, err)
		}
	} else if node.Role == k3d.ServerRole {
		if err := patchServerSpec(node, runtime); err != nil {
			return fmt.Errorf("failed to patch server spec on node %s: %w", node.Name, err)
		}
	}

	if _, anyFixEnabled := fixes.GetFixes(runtime); anyFixEnabled {
		node.K3dEntrypoint = true
	}

	// memory limits
	if node.Memory != "" {
		if runtime != runtimes.Docker {
			l.Log().Warn("ignoring specified memory limits as runtime is not Docker")
		} else {
			memory, err := dockerunits.RAMInBytes(node.Memory)
			if err != nil {
				return fmt.Errorf("invalid memory limit format: %w", err)
			}
			// mount fake meminfo as readonly
			fakemempath, err := util.MakeFakeMeminfo(memory, node.Name)
			if err != nil {
				return fmt.Errorf("failed to create fake meminfo: %w", err)
			}
			node.Volumes = append(node.Volumes, fmt.Sprintf("%s:%s:ro", fakemempath, util.MemInfoPath))
			// mount empty edac folder, but only if it exists
			exists, err := docker.CheckIfDirectoryExists(ctx, node.Image, util.EdacFolderPath)
			if err != nil {
				return fmt.Errorf("failed to check for the existence of edac folder: %w", err)
			}
			if exists {
				l.Log().Debugln("Found edac folder")
				fakeedacpath, err := util.MakeFakeEdac(node.Name)
				if err != nil {
					return fmt.Errorf("failed to create fake edac: %w", err)
				}
				node.Volumes = append(node.Volumes, fmt.Sprintf("%s:%s:ro", fakeedacpath, util.EdacFolderPath))
			}
		}
	}

	/*
	 * CREATION
	 */
	if err := runtime.CreateNode(ctx, node); err != nil {
		return fmt.Errorf("runtime failed to create node '%s': %w", node.Name, err)
	}

	return nil
}

// NodeDelete deletes an existing node
func NodeDelete(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, opts k3d.NodeDeleteOpts) error {
	// delete node
	if err := runtime.DeleteNode(ctx, node); err != nil {
		l.Log().Error(err)
	}

	// delete fake folder created for limits
	if node.Memory != "" {
		l.Log().Debug("Cleaning fake files folder from k3d config dir for this node...")
		filepath, err := util.GetNodeFakerDirOrCreate(node.Name)
		if err != nil {
			l.Log().Errorf("Could not get fake files folder for node %s: %+v", node.Name, err)
		}
		err = os.RemoveAll(filepath)
		if err != nil {
			// this error prob should not be fatal, just log it
			l.Log().Errorf("Could not remove fake files folder for node %s: %+v", node.Name, err)
		}
	}

	// update the server loadbalancer
	if !opts.SkipLBUpdate && (node.Role == k3d.ServerRole || node.Role == k3d.AgentRole) {
		cluster, err := ClusterGet(ctx, runtime, &k3d.Cluster{Name: node.RuntimeLabels[k3d.LabelClusterName]})
		if err != nil {
			return fmt.Errorf("failed fo find cluster for node '%s': %w", node.Name, err)
		}

		// if it's a server node, then update the loadbalancer configuration
		if node.Role == k3d.ServerRole && cluster.ServerLoadBalancer != nil {
			if err := UpdateLoadbalancerConfig(ctx, runtime, cluster); err != nil {
				if !errors.Is(err, ErrLBConfigHostNotFound) {
					return fmt.Errorf("failed to update cluster loadbalancer: %w", err)
				}
			}
		}
	}

	return nil
}

// patchAgentSpec adds agent node specific settings to a node
func patchAgentSpec(node *k3d.Node) error {
	if node.Cmd == nil {
		node.Cmd = []string{"agent"}
	}

	return nil
}

// patchServerSpec adds server node specific settings to a node
func patchServerSpec(node *k3d.Node, runtime runtimes.Runtime) error {
	// command / arguments
	if node.Cmd == nil {
		node.Cmd = []string{"server"}
	}

	// Add labels and TLS SAN for the exposed API
	// FIXME: For now, the labels concerning the API on the server nodes are only being used for configuring the kubeconfig
	node.RuntimeLabels[k3d.LabelServerAPIHostIP] = node.ServerOpts.KubeAPI.Binding.HostIP // TODO: maybe get docker machine IP here
	node.RuntimeLabels[k3d.LabelServerAPIHost] = node.ServerOpts.KubeAPI.Host
	node.RuntimeLabels[k3d.LabelServerAPIPort] = node.ServerOpts.KubeAPI.Binding.HostPort

	node.Args = append(node.Args, "--tls-san", node.RuntimeLabels[k3d.LabelServerAPIHost]) // add TLS SAN for non default host name
	if node.RuntimeLabels[k3d.LabelServerLoadBalancer] != "" {
		// add TLS SAN for server loadbalancer
		node.Args = append(node.Args, "--tls-san", node.RuntimeLabels[k3d.LabelServerLoadBalancer])
	}

	return nil
}

// Check if node associated with K3S URL value is still active in the cluster
func checkK3SURLIsActive(nodes []*k3d.Node, k3sURL string) bool {
	// extract the node name
	re := regexp.MustCompile("K3S_URL=https?://([a-zA-Z0-9_.-]+)")
	match := re.FindStringSubmatch(k3sURL)

	if match == nil {
		return false
	}

	for _, node := range nodes {
		if node.Name == match[1] {
			return true
		}
	}

	return false
}

// NodeList returns a list of all existing clusters
func NodeList(ctx context.Context, runtime runtimes.Runtime) ([]*k3d.Node, error) {
	nodes, err := runtime.GetNodesByLabel(ctx, k3d.DefaultRuntimeLabels)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	return nodes, nil
}

// NodeGet returns a node matching the specified node fields
func NodeGet(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node) (*k3d.Node, error) {
	// get node
	node, err := runtime.GetNode(ctx, node)
	if err != nil {
		return nil, fmt.Errorf("failed to get node '%s': %w", node.Name, err)
	}

	return node, nil
}

// NodeWaitForLogMessage follows the logs of a node container and returns if it finds a specific line in there (or timeout is reached)
func NodeWaitForLogMessage(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, message string, since time.Time) error {
	l.Log().Tracef("NodeWaitForLogMessage: Node '%s' waiting for log message '%s' since '%+v'", node.Name, message, since)

	message = strings.ToLower(message)

	// specify max number of retries if container is in crashloop (as defined by last seen message being a fatal log)
	backOffLimit := k3d.DefaultNodeWaitForLogMessageCrashLoopBackOffLimit
	if l, ok := os.LookupEnv(k3d.K3dEnvDebugNodeWaitBackOffLimit); ok {
		limit, err := strconv.Atoi(l)
		if err == nil {
			backOffLimit = limit
		}
	}

	// start a goroutine to print a warning continuously if a node is restarting for quite some time already
	donechan := make(chan struct{})
	defer close(donechan)
	go func(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, since time.Time, donechan chan struct{}) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-donechan:
				return
			default:
			}
			// check if the container is restarting
			running, status, _ := runtime.GetNodeStatus(ctx, node)
			if running && status == k3d.NodeStatusRestarting && time.Since(since) > k3d.NodeWaitForLogMessageRestartWarnTime {
				l.Log().Warnf("Node '%s' is restarting for more than %s now. Possibly it will recover soon (e.g. when it's waiting to join). Consider using a creation timeout to avoid waiting forever in a Restart Loop.", node.Name, k3d.NodeWaitForLogMessageRestartWarnTime)
			}
			time.Sleep(500 * time.Millisecond)
		}
	}(ctx, runtime, node, since, donechan)

	// pre-building error message in case the node stops returning logs for some reason: to be enriched with scanner error
	errMsg := fmt.Errorf("error waiting for log line `%s` from node '%s': stopped returning log lines", message, node.Name)

	// Start loop to check log stream for specified log message.
	// We're looping here, as sometimes the containers run into a crash loop, but *may* recover from that
	// e.g. when a new server is joining an existing cluster and has to wait for another member to finish learning.
	// The logstream returned by docker ends everytime the container restarts, so we have to start from the beginning.
	for i := 0; i < backOffLimit; i++ {
		// get the log stream (reader is following the logstream)
		out, err := runtime.GetNodeLogs(ctx, node, since, &runtimeTypes.NodeLogsOpts{Follow: true})
		if out != nil {
			defer out.Close()
		}
		if err != nil {
			return fmt.Errorf("Failed waiting for log message '%s' from node '%s': %w", message, node.Name, err)
		}

		// We're scanning the logstream continuously line-by-line
		scanner := bufio.NewScanner(out)
		var previousline string

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				if ctx.Err() == context.DeadlineExceeded {
					d, ok := ctx.Deadline()
					if ok {
						l.Log().Debugf("NodeWaitForLogMessage: Context Deadline (%s) > Current Time (%s)", d, time.Now())
					}
					return fmt.Errorf("Context deadline exceeded while waiting for log message '%s' of node %s: %w", message, node.Name, ctx.Err())
				}
				return ctx.Err()
			default:
			}

			if strings.Contains(os.Getenv(k3d.K3dEnvLogNodeWaitLogs), string(node.Role)) {
				l.Log().Tracef(">>> Parsing log line: `%s`", scanner.Text())
			}
			// check if we can find the specified line in the log
			if strings.Contains(strings.ToLower(scanner.Text()), message) {
				l.Log().Tracef("Found target message `%s` in log line `%s`", message, scanner.Text())
				l.Log().Debugf("Finished waiting for log message '%s' from node '%s'", message, node.Name)
				return nil
			}

			previousline = scanner.Text()
		}

		if e := scanner.Err(); e != nil {
			errMsg = fmt.Errorf("%v: %w", errMsg, e)
		}

		out.Close() // no more input on scanner, but target log not yet found -> close current logreader (precautionary)

		// we got here, because the logstream ended (no more input on scanner), so we check if maybe the container crashed
		if strings.Contains(previousline, "level=fatal") {
			// case 1: last log line we saw contained a fatal error, so probably it crashed and we want to retry on restart
			l.Log().Warnf("warning: encountered fatal log from node %s (retrying %d/%d): %s", node.Name, i, backOffLimit, previousline)
			out.Close()
			time.Sleep(500 * time.Millisecond)
			continue
		} else {
			l.Log().Tracef("Non-fatal last log line in node %s: %s", node.Name, previousline)
			// case 2: last log line we saw did not contain a fatal error, so we break the loop here and return a generic error
			break
		}
	}

	running, status, err := runtime.GetNodeStatus(ctx, node)
	if err == nil {
		return fmt.Errorf("%v: node %s is running=%v in status=%s", errMsg, node.Name, running, status)
	}
	return errMsg
}

// NodeFilterByRoles filters a list of nodes by their roles
func NodeFilterByRoles(nodes []*k3d.Node, includeRoles, excludeRoles []k3d.Role) []*k3d.Node {
	// check for conflicting filters
	for _, includeRole := range includeRoles {
		for _, excludeRole := range excludeRoles {
			if includeRole == excludeRole {
				l.Log().Warnf("You've specified the same role ('%s') for inclusion and exclusion. Exclusion precedes inclusion.", includeRole)
			}
		}
	}

	resultList := []*k3d.Node{}
nodeLoop:
	for _, node := range nodes {
		// exclude > include
		for _, excludeRole := range excludeRoles {
			if node.Role == excludeRole {
				continue nodeLoop
			}
		}

		// include < exclude
		for _, includeRole := range includeRoles {
			if node.Role == includeRole {
				resultList = append(resultList, node)
				continue nodeLoop
			}
		}
	}

	l.Log().Tracef("Filteres %d nodes by roles (in: %+v | ex: %+v), got %d left", len(nodes), includeRoles, excludeRoles, len(resultList))

	return resultList
}

// NodeEdit let's you update an existing node
func NodeEdit(ctx context.Context, runtime runtimes.Runtime, existingNode, changeset *k3d.Node) error {
	/*
	 * Make a deep copy of the existing node
	 */

	result, err := CopyNode(ctx, existingNode, CopyNodeOpts{keepState: false})
	if err != nil {
		return fmt.Errorf("failed to copy node %s: %w", existingNode.Name, err)
	}

	/*
	 * Apply changes
	 */

	// === Ports ===
	if result.Ports == nil {
		result.Ports = nat.PortMap{}
	}
	for port, portbindings := range changeset.Ports {
	loopChangesetPortbindings:
		for _, portbinding := range portbindings {
			// loop over existing portbindings to avoid port collisions (docker doesn't check for it)
			for _, existingPB := range result.Ports[port] {
				if util.IsPortBindingEqual(portbinding, existingPB) { // also matches on "equal" HostIPs (127.0.0.1, "", 0.0.0.0)
					l.Log().Tracef("Skipping existing PortBinding: %+v", existingPB)
					continue loopChangesetPortbindings
				}
			}
			l.Log().Tracef("Adding portbinding %+v for port %s", portbinding, port.Port())
			result.Ports[port] = append(result.Ports[port], portbinding)
		}
	}

	// --- Loadbalancer specifics ---
	if result.Role == k3d.LoadBalancerRole {
		cluster, err := ClusterGet(ctx, runtime, &k3d.Cluster{Name: existingNode.RuntimeLabels[k3d.LabelClusterName]})
		if err != nil {
			return fmt.Errorf("error updating loadbalancer config: %w", err)
		}
		if cluster.ServerLoadBalancer == nil {
			cluster.ServerLoadBalancer = k3d.NewLoadbalancer()
		}
		cluster.ServerLoadBalancer.Node = result
		lbConfig, err := LoadbalancerGenerateConfig(cluster)
		if err != nil {
			return fmt.Errorf("error generating loadbalancer config: %v", err)
		}

		// prepare to write config to lb container
		configyaml, err := yaml.Marshal(lbConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal loadbalancer config: %w", err)
		}

		writeLbConfigAction := k3d.NodeHook{
			Stage: k3d.LifecycleStagePreStart,
			Action: actions.WriteFileAction{
				Runtime:     runtime,
				Dest:        k3d.DefaultLoadbalancerConfigPath,
				Mode:        0744,
				Content:     configyaml,
				Description: "Write Loadbalancer Configuration",
			},
		}

		result.HookActions = append(result.HookActions, writeLbConfigAction)
	}

	// replace existing node
	return NodeReplace(ctx, runtime, existingNode, result)
}

func NodeReplace(ctx context.Context, runtime runtimes.Runtime, old, new *k3d.Node) error {
	// rename existing node
	oldNameTemp := fmt.Sprintf("%s-%s", old.Name, util.GenerateRandomString(5))
	oldNameOriginal := old.Name
	l.Log().Infof("Renaming existing node %s to %s...", old.Name, oldNameTemp)
	if err := runtime.RenameNode(ctx, old, oldNameTemp); err != nil {
		return fmt.Errorf("runtime failed to rename node '%s': %w", old.Name, err)
	}
	old.Name = oldNameTemp

	// create (not start) new node
	l.Log().Infof("Creating new node %s...", new.Name)
	if err := NodeCreate(ctx, runtime, new, k3d.NodeCreateOpts{Wait: true}); err != nil {
		if err := runtime.RenameNode(ctx, old, oldNameOriginal); err != nil {
			return fmt.Errorf("Failed to create new node. Also failed to rename %s back to %s: %+v", old.Name, oldNameOriginal, err)
		}
		return fmt.Errorf("Failed to create new node. Brought back old node: %+v", err)
	}

	// stop existing/old node
	l.Log().Infof("Stopping existing node %s...", old.Name)
	if err := runtime.StopNode(ctx, old); err != nil {
		return fmt.Errorf("runtime failed to stop node '%s': %w", old.Name, err)
	}

	// start new node
	l.Log().Infof("Starting new node %s...", new.Name)
	if err := NodeStart(ctx, runtime, new, &k3d.NodeStartOpts{Wait: true, NodeHooks: new.HookActions}); err != nil {
		if err := NodeDelete(ctx, runtime, new, k3d.NodeDeleteOpts{SkipLBUpdate: true}); err != nil {
			return fmt.Errorf("Failed to start new node. Also failed to rollback: %+v", err)
		}
		if err := runtime.RenameNode(ctx, old, oldNameOriginal); err != nil {
			return fmt.Errorf("Failed to start new node. Also failed to rename %s back to %s: %+v", old.Name, oldNameOriginal, err)
		}
		old.Name = oldNameOriginal
		if err := NodeStart(ctx, runtime, old, &k3d.NodeStartOpts{Wait: true}); err != nil {
			return fmt.Errorf("Failed to start new node. Also failed to restart old node: %+v", err)
		}
		return fmt.Errorf("Failed to start new node. Rolled back: %+v", err)
	}

	// cleanup: delete old node
	l.Log().Infof("Deleting old node %s...", old.Name)
	if err := NodeDelete(ctx, runtime, old, k3d.NodeDeleteOpts{SkipLBUpdate: true}); err != nil {
		return fmt.Errorf("failed to delete old node '%s': %w", old.Name, err)
	}

	// done
	return nil
}

type CopyNodeOpts struct {
	keepState bool
}

func CopyNode(ctx context.Context, src *k3d.Node, opts CopyNodeOpts) (*k3d.Node, error) {
	targetCopy, err := copystruct.Copy(src)
	if err != nil {
		return nil, fmt.Errorf("failed to copy node struct: %w", err)
	}

	result := targetCopy.(*k3d.Node)

	if !opts.keepState {
		// ensure that node state is empty
		result.State = k3d.NodeState{}
	}

	return result, nil
}
