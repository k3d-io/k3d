/*
Copyright © 2020-2021 The k3d Author(s)

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
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"time"

	copystruct "github.com/mitchellh/copystructure"
	"gopkg.in/yaml.v2"

	"github.com/docker/go-connections/nat"
	dockerunits "github.com/docker/go-units"
	"github.com/imdario/mergo"
	"github.com/rancher/k3d/v4/pkg/actions"
	"github.com/rancher/k3d/v4/pkg/runtimes"
	"github.com/rancher/k3d/v4/pkg/runtimes/docker"
	runtimeErrors "github.com/rancher/k3d/v4/pkg/runtimes/errors"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	"github.com/rancher/k3d/v4/pkg/types/fixes"
	"github.com/rancher/k3d/v4/pkg/util"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// NodeAddToCluster adds a node to an existing cluster
func NodeAddToCluster(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, cluster *k3d.Cluster, createNodeOpts k3d.NodeCreateOpts) error {
	targetClusterName := cluster.Name
	cluster, err := ClusterGet(ctx, runtime, cluster)
	if err != nil {
		log.Errorf("Failed to find specified cluster '%s'", targetClusterName)
		return err
	}

	// network
	node.Networks = []string{cluster.Network.Name}

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
		log.Debugf("Didn't find node with role '%s' in cluster '%s'. Choosing any other node (and using defaults)...", node.Role, cluster.Name)
		node.Cmd = k3d.DefaultRoleCmds[node.Role]
		for _, existingNode := range cluster.Nodes {
			if existingNode.Role != k3d.LoadBalancerRole { // any role except for the LoadBalancer role
				srcNode = existingNode
				break
			}
		}
	}

	// get node details
	srcNode, err = NodeGet(ctx, runtime, srcNode)
	if err != nil {
		return err
	}

	/*
	 * Sanitize Source Node
	 * -> remove fields that are not safe to copy as they break something down the stream
	 */

	// TODO: I guess proper deduplication can be handled in a cleaner/better way or at the infofaker level at some point
	for _, forbiddenMount := range util.DoNotCopyVolumeSuffices {
		for i, mount := range node.Volumes {
			if strings.Contains(mount, forbiddenMount) {
				log.Tracef("Dropping copied volume mount %s to avoid issues...", mount)
				node.Volumes = util.RemoveElementFromStringSlice(node.Volumes, i)
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
					log.Tracef("Dropping '%s' from source node's cmd", forbiddenCmd)
					srcNode.Cmd = append(srcNode.Cmd[:i], srcNode.Cmd[i+1:]...)
				}
			}
			for i, arg := range node.Args {
				// cut out the '--cluster-init' flag as this should only be done by the initializing server node
				if arg == forbiddenCmd {
					log.Tracef("Dropping '%s' from source node's args", forbiddenCmd)
					srcNode.Args = append(srcNode.Args[:i], srcNode.Args[i+1:]...)
				}
			}
		}
	}

	log.Debugf("Adding node %s to cluster %s based on existing (sanitized) node %s", node.Name, cluster.Name, srcNode.Name)
	log.Tracef("Sanitized Source Node: %+v\nNew Node: %+v", srcNode, node)

	// fetch registry config
	registryConfigBytes := []byte{}
	registryConfigReader, err := runtime.ReadFromNode(ctx, k3d.DefaultRegistriesFilePath, srcNode)
	if err != nil {
		if !errors.Is(err, runtimeErrors.ErrRuntimeFileNotFound) {
			log.Warnf("Failed to read registry config from node %s: %+v", node.Name, err)
		}
	} else {
		defer registryConfigReader.Close()

		var err error
		registryConfigBytes, err = ioutil.ReadAll(registryConfigReader)
		if err != nil {
			log.Warnf("Failed to read registry config from node %s: %+v", node.Name, err)
		}
		registryConfigReader.Close()
		registryConfigBytes = bytes.Trim(registryConfigBytes[512:], "\x00") // trim control characters, etc.
	}

	// merge node config of new node into existing node config
	if err := mergo.MergeWithOverwrite(srcNode, *node); err != nil {
		log.Errorln("Failed to merge new node config into existing node config")
		return err
	}

	node = srcNode

	log.Debugf("Resulting node %+v", node)

	k3sURLFound := false
	for _, envVar := range node.Env {
		if strings.HasPrefix(envVar, "K3S_URL") {
			k3sURLFound = true
			break
		}
	}
	if !k3sURLFound {
		if url, ok := node.RuntimeLabels[k3d.LabelClusterURL]; ok {
			node.Env = append(node.Env, fmt.Sprintf("K3S_URL=%s", url))
		} else {
			log.Warnln("Failed to find K3S_URL value!")
		}
	}

	// add node actions
	if len(registryConfigBytes) != 0 {
		if createNodeOpts.NodeHooks == nil {
			createNodeOpts.NodeHooks = []k3d.NodeHook{}
		}
		createNodeOpts.NodeHooks = append(createNodeOpts.NodeHooks, k3d.NodeHook{
			Stage: k3d.LifecycleStagePreStart,
			Action: actions.WriteFileAction{
				Runtime: runtime,
				Content: registryConfigBytes,
				Dest:    k3d.DefaultRegistriesFilePath,
				Mode:    0644,
			},
		})
	}

	// clear status fields
	node.State.Running = false
	node.State.Status = ""

	if err := NodeRun(ctx, runtime, node, createNodeOpts); err != nil {
		return err
	}

	// if it's a server node, then update the loadbalancer configuration
	if node.Role == k3d.ServerRole {
		log.Infoln("Updating loadbalancer config to include new server node(s)")
		if err := UpdateLoadbalancerConfig(ctx, runtime, cluster); err != nil {
			if !errors.Is(err, LBConfigErrHostNotFound) {
				return fmt.Errorf("error updating loadbalancer: %w", err)
			}
		}
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

	nodeWaitGroup, ctx := errgroup.WithContext(ctx)
	for _, node := range nodes {
		currentNode := node
		nodeWaitGroup.Go(func() error {
			return NodeAddToCluster(ctx, runtime, currentNode, cluster, createNodeOpts)
		})
	}
	if err := nodeWaitGroup.Wait(); err != nil {
		return fmt.Errorf("Failed to add one or more nodes: %w", err)
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
			log.Error(err)
		}
		if createNodeOpts.Wait {
			currentNode := node
			nodeWaitGroup.Go(func() error {
				log.Debugf("Starting to wait for node '%s'", currentNode.Name)
				readyLogMessage := k3d.ReadyLogMessageByRole[currentNode.Role]
				if readyLogMessage != "" {
					return NodeWaitForLogMessage(ctx, runtime, currentNode, readyLogMessage, time.Time{})
				}
				log.Warnf("NodeCreateMulti: Set to wait for node %s to get ready, but there's no target log message defined", currentNode.Name)
				return nil
			})
		}
	}

	if err := nodeWaitGroup.Wait(); err != nil {
		log.Errorln("Failed to bring up all nodes in time. Check the logs:")
		log.Errorf(">>> %+v", err)
		return fmt.Errorf("Failed to create nodes")
	}

	return nil
}

// NodeRun creates and starts a node
func NodeRun(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, nodeCreateOpts k3d.NodeCreateOpts) error {
	if err := NodeCreate(ctx, runtime, node, nodeCreateOpts); err != nil {
		return err
	}

	if err := NodeStart(ctx, runtime, node, k3d.NodeStartOpts{
		Wait:      nodeCreateOpts.Wait,
		Timeout:   nodeCreateOpts.Timeout,
		NodeHooks: nodeCreateOpts.NodeHooks,
	}); err != nil {
		return err
	}

	return nil
}

// NodeStart starts an existing node
func NodeStart(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, nodeStartOpts k3d.NodeStartOpts) error {

	// return early, if the node is already running
	if node.State.Running {
		log.Infof("Node %s is already running", node.Name)
		return nil
	}

	// FIXME: FixCgroupV2 - to be removed when fixed upstream
	if node.Role == k3d.ServerRole || node.Role == k3d.AgentRole {
		EnableCgroupV2FixIfNeeded(runtime)
		if fixes.FixCgroupV2Enabled() {

			if nodeStartOpts.NodeHooks == nil {
				nodeStartOpts.NodeHooks = []k3d.NodeHook{}
			}

			nodeStartOpts.NodeHooks = append(nodeStartOpts.NodeHooks, k3d.NodeHook{
				Stage: k3d.LifecycleStagePreStart,
				Action: actions.WriteFileAction{
					Runtime: runtime,
					Content: fixes.CgroupV2Entrypoint,
					Dest:    "/bin/entrypoint.sh",
					Mode:    0744,
				},
			})
		}
	}

	startTime := time.Now()
	log.Debugf("Node %s Start Time: %+v", node.Name, startTime)

	// execute lifecycle hook actions
	for _, hook := range nodeStartOpts.NodeHooks {
		if hook.Stage == k3d.LifecycleStagePreStart {
			log.Tracef("Node %s: Executing preStartAction '%s'", node.Name, reflect.TypeOf(hook))
			if err := hook.Action.Run(ctx, node); err != nil {
				log.Errorf("Node %s: Failed executing preStartAction '%+v': %+v", node.Name, hook, err)
			}
		}
	}

	// start the node
	log.Tracef("Starting node '%s'", node.Name)

	if err := runtime.StartNode(ctx, node); err != nil {
		log.Errorf("Failed to start node '%s'", node.Name)
		return err
	}

	if node.State.Started != "" {
		ts, err := time.Parse("2006-01-02T15:04:05.999999999Z", node.State.Started)
		if err != nil {
			log.Debugf("Failed to parse '%s.State.Started' timestamp '%s', falling back to calulated time", node.Name, node.State.Started)
		}
		startTime = ts.Truncate(time.Second)
		log.Debugf("Truncated %s to %s", ts, startTime)
	}

	if nodeStartOpts.Wait {
		if nodeStartOpts.ReadyLogMessage == "" {
			nodeStartOpts.ReadyLogMessage = k3d.ReadyLogMessageByRole[node.Role]
		}
		if nodeStartOpts.ReadyLogMessage != "" {
			log.Debugf("Waiting for node %s to get ready (Log: '%s')", node.Name, nodeStartOpts.ReadyLogMessage)
			if err := NodeWaitForLogMessage(ctx, runtime, node, nodeStartOpts.ReadyLogMessage, startTime); err != nil {
				return fmt.Errorf("Node %s failed to get ready: %+v", node.Name, err)
			}
		} else {
			log.Warnf("NodeStart: Set to wait for node %s to be ready, but there's no target log message defined", node.Name)
		}
	}

	return nil
}

// NodeCreate creates a new containerized k3s node
func NodeCreate(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, createNodeOpts k3d.NodeCreateOpts) error {
	// FIXME: FixCgroupV2 - to be removed when fixed upstream
	EnableCgroupV2FixIfNeeded(runtime)
	log.Tracef("Creating node from spec\n%+v", node)

	/*
	 * CONFIGURATION
	 */

	/* global node configuration (applies for any node role) */

	// ### Labels ###
	labels := make(map[string]string)
	for k, v := range k3d.DefaultRuntimeLabels {
		labels[k] = v
	}
	for k, v := range k3d.DefaultRuntimeLabelsVar {
		labels[k] = v
	}
	for k, v := range node.RuntimeLabels {
		labels[k] = v
	}
	node.RuntimeLabels = labels
	// second most important: the node role label
	node.RuntimeLabels[k3d.LabelRole] = string(node.Role)

	for k, v := range node.K3sNodeLabels {
		node.Args = append(node.Args, "--node-label", fmt.Sprintf("%s=%s", k, v))
	}

	// ### Environment ###
	node.Env = append(node.Env, k3d.DefaultNodeEnv...) // append default node env vars

	// specify options depending on node role
	if node.Role == k3d.AgentRole { // TODO: check here AND in CLI or only here?
		if err := patchAgentSpec(node); err != nil {
			return err
		}
	} else if node.Role == k3d.ServerRole {
		if err := patchServerSpec(node, runtime); err != nil {
			return err
		}
	}

	// memory limits
	if node.Memory != "" {
		if runtime != runtimes.Docker {
			log.Warn("ignoring specified memory limits as runtime is not Docker")
		} else {
			memory, err := dockerunits.RAMInBytes(node.Memory)
			if err != nil {
				return fmt.Errorf("Invalid memory limit format: %+v", err)
			}
			// mount fake meminfo as readonly
			fakemempath, err := util.MakeFakeMeminfo(memory, node.Name)
			if err != nil {
				return fmt.Errorf("Failed to create fake meminfo: %+v", err)
			}
			node.Volumes = append(node.Volumes, fmt.Sprintf("%s:%s:ro", fakemempath, util.MemInfoPath))
			// mount empty edac folder, but only if it exists
			exists, err := docker.CheckIfDirectoryExists(ctx, node.Image, util.EdacFolderPath)
			if err != nil {
				return fmt.Errorf("Failed to check for the existence of edac folder: %+v", err)
			}
			if exists {
				log.Debugln("Found edac folder")
				fakeedacpath, err := util.MakeFakeEdac(node.Name)
				if err != nil {
					return fmt.Errorf("Failed to create fake edac: %+v", err)
				}
				node.Volumes = append(node.Volumes, fmt.Sprintf("%s:%s:ro", fakeedacpath, util.EdacFolderPath))
			}
		}
	}

	/*
	 * CREATION
	 */
	if err := runtime.CreateNode(ctx, node); err != nil {
		return err
	}

	return nil
}

// NodeDelete deletes an existing node
func NodeDelete(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, opts k3d.NodeDeleteOpts) error {
	// delete node
	if err := runtime.DeleteNode(ctx, node); err != nil {
		log.Error(err)
	}

	// delete fake folder created for limits
	if node.Memory != "" {
		log.Debug("Cleaning fake files folder from k3d config dir for this node...")
		filepath, err := util.GetNodeFakerDirOrCreate(node.Name)
		err = os.RemoveAll(filepath)
		if err != nil {
			// this err prob should not be fatal, just log it
			log.Errorf("Could not remove fake files folder for node %s: %+v", node.Name, err)
		}
	}

	// update the server loadbalancer
	if !opts.SkipLBUpdate && (node.Role == k3d.ServerRole || node.Role == k3d.AgentRole) {
		cluster, err := ClusterGet(ctx, runtime, &k3d.Cluster{Name: node.RuntimeLabels[k3d.LabelClusterName]})
		if err != nil {
			log.Errorf("Failed to find cluster for node '%s'", node.Name)
			return err
		}

		// if it's a server node, then update the loadbalancer configuration
		if node.Role == k3d.ServerRole {
			if err := UpdateLoadbalancerConfig(ctx, runtime, cluster); err != nil {
				if !errors.Is(err, LBConfigErrHostNotFound) {
					return fmt.Errorf("Failed to update cluster loadbalancer: %w", err)
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

	// If the runtime is docker, attempt to use the docker host
	if runtime == runtimes.Docker {
		dockerHost := runtime.GetHost()
		if dockerHost != "" {
			dockerHost = strings.Split(dockerHost, ":")[0] // remove the port
			log.Tracef("Using docker host %s", dockerHost)
			node.RuntimeLabels[k3d.LabelServerAPIHostIP] = dockerHost
			node.RuntimeLabels[k3d.LabelServerAPIHost] = dockerHost
		}
	}

	node.Args = append(node.Args, "--tls-san", node.RuntimeLabels[k3d.LabelServerAPIHost]) // add TLS SAN for non default host name

	return nil
}

// NodeList returns a list of all existing clusters
func NodeList(ctx context.Context, runtime runtimes.Runtime) ([]*k3d.Node, error) {
	nodes, err := runtime.GetNodesByLabel(ctx, k3d.DefaultRuntimeLabels)
	if err != nil {
		log.Errorln("Failed to get nodes")
		return nil, err
	}

	return nodes, nil
}

// NodeGet returns a node matching the specified node fields
func NodeGet(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node) (*k3d.Node, error) {
	// get node
	node, err := runtime.GetNode(ctx, node)
	if err != nil {
		log.Errorf("Failed to get node '%s'", node.Name)
		return nil, err
	}

	return node, nil
}

// NodeWaitForLogMessage follows the logs of a node container and returns if it finds a specific line in there (or timeout is reached)
func NodeWaitForLogMessage(ctx context.Context, runtime runtimes.Runtime, node *k3d.Node, message string, since time.Time) error {
	log.Tracef("NodeWaitForLogMessage: Node '%s' waiting for log message '%s' since '%+v'", node.Name, message, since)
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				d, ok := ctx.Deadline()
				if ok {
					log.Debugf("NodeWaitForLogMessage: Context Deadline (%s) > Current Time (%s)", d, time.Now())
				}
				return fmt.Errorf("Context deadline exceeded while waiting for log message '%s' of node %s: %w", message, node.Name, ctx.Err())
			}
			return ctx.Err()
		default:
		}

		// read the logs
		out, err := runtime.GetNodeLogs(ctx, node, since)
		if err != nil {
			if out != nil {
				out.Close()
			}
			return fmt.Errorf("Failed waiting for log message '%s' from node '%s': %w", message, node.Name, err)
		}
		defer out.Close()

		buf := new(bytes.Buffer)
		nRead, _ := buf.ReadFrom(out)
		out.Close()
		output := buf.String()

		if nRead > 0 && strings.Contains(os.Getenv("K3D_LOG_NODE_WAIT_LOGS"), string(node.Role)) {
			log.Tracef("=== Read logs since %s ===\n%s\n", since, output)
		}
		// check if we can find the specified line in the log
		if nRead > 0 && strings.Contains(output, message) {
			if log.GetLevel() >= log.TraceLevel {
				temp := strings.Split(output, "\n")
				for _, l := range temp {
					if strings.Contains(l, message) {
						log.Tracef("Found target log line: `%s`", l)
					}
				}
			}
			break
		}

		// check if the container is restarting
		running, status, _ := runtime.GetNodeStatus(ctx, node)
		if running && status == k3d.NodeStatusRestarting && time.Now().Sub(since) > k3d.NodeWaitForLogMessageRestartWarnTime {
			log.Warnf("Node '%s' is restarting for more than a minute now. Possibly it will recover soon (e.g. when it's waiting to join). Consider using a creation timeout to avoid waiting forever in a Restart Loop.", node.Name)
		}

		time.Sleep(500 * time.Millisecond) // wait for half a second to avoid overloading docker (error `socket: too many open files`)
	}
	log.Debugf("Finished waiting for log message '%s' from node '%s'", message, node.Name)
	return nil
}

// NodeFilterByRoles filters a list of nodes by their roles
func NodeFilterByRoles(nodes []*k3d.Node, includeRoles, excludeRoles []k3d.Role) []*k3d.Node {
	// check for conflicting filters
	for _, includeRole := range includeRoles {
		for _, excludeRole := range excludeRoles {
			if includeRole == excludeRole {
				log.Warnf("You've specified the same role ('%s') for inclusion and exclusion. Exclusion precedes inclusion.", includeRole)
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

	log.Tracef("Filteres %d nodes by roles (in: %+v | ex: %+v), got %d left", len(nodes), includeRoles, excludeRoles, len(resultList))

	return resultList
}

// NodeEdit let's you update an existing node
func NodeEdit(ctx context.Context, runtime runtimes.Runtime, existingNode, changeset *k3d.Node) error {

	/*
	 * Make a deep copy of the existing node
	 */

	result, err := CopyNode(ctx, existingNode, CopyNodeOpts{keepState: false})
	if err != nil {
		return err
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
					log.Tracef("Skipping existing PortBinding: %+v", existingPB)
					continue loopChangesetPortbindings
				}
			}
			log.Tracef("Adding portbinding %+v for port %s", portbinding, port.Port())
			result.Ports[port] = append(result.Ports[port], portbinding)
		}
	}

	// --- Loadbalancer specifics ---
	if result.Role == k3d.LoadBalancerRole {
		cluster, err := ClusterGet(ctx, runtime, &k3d.Cluster{Name: existingNode.RuntimeLabels[k3d.LabelClusterName]})
		if err != nil {
			return fmt.Errorf("error updating loadbalancer config: %w", err)
		}
		cluster.ServerLoadBalancer = result
		lbConfig, err := LoadbalancerGenerateConfig(cluster)
		if err != nil {
			return fmt.Errorf("error generating loadbalancer config: %v", err)
		}

		// prepare to write config to lb container
		configyaml, err := yaml.Marshal(lbConfig)
		if err != nil {
			return err
		}

		writeLbConfigAction := k3d.NodeHook{
			Stage: k3d.LifecycleStagePreStart,
			Action: actions.WriteFileAction{
				Runtime: runtime,
				Dest:    k3d.DefaultLoadbalancerConfigPath,
				Mode:    0744,
				Content: configyaml,
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
	log.Infof("Renaming existing node %s to %s...", old.Name, oldNameTemp)
	if err := runtime.RenameNode(ctx, old, oldNameTemp); err != nil {
		return err
	}
	old.Name = oldNameTemp

	// create (not start) new node
	log.Infof("Creating new node %s...", new.Name)
	if err := NodeCreate(ctx, runtime, new, k3d.NodeCreateOpts{Wait: true}); err != nil {
		if err := runtime.RenameNode(ctx, old, oldNameOriginal); err != nil {
			return fmt.Errorf("Failed to create new node. Also failed to rename %s back to %s: %+v", old.Name, oldNameOriginal, err)
		}
		return fmt.Errorf("Failed to create new node. Brought back old node: %+v", err)
	}

	// stop existing/old node
	log.Infof("Stopping existing node %s...", old.Name)
	if err := runtime.StopNode(ctx, old); err != nil {
		return err
	}

	// start new node
	log.Infof("Starting new node %s...", new.Name)
	if err := NodeStart(ctx, runtime, new, k3d.NodeStartOpts{Wait: true, NodeHooks: new.HookActions}); err != nil {
		if err := NodeDelete(ctx, runtime, new, k3d.NodeDeleteOpts{SkipLBUpdate: true}); err != nil {
			return fmt.Errorf("Failed to start new node. Also failed to rollback: %+v", err)
		}
		if err := runtime.RenameNode(ctx, old, oldNameOriginal); err != nil {
			return fmt.Errorf("Failed to start new node. Also failed to rename %s back to %s: %+v", old.Name, oldNameOriginal, err)
		}
		old.Name = oldNameOriginal
		if err := NodeStart(ctx, runtime, old, k3d.NodeStartOpts{Wait: true}); err != nil {
			return fmt.Errorf("Failed to start new node. Also failed to restart old node: %+v", err)
		}
		return fmt.Errorf("Failed to start new node. Rolled back: %+v", err)
	}

	// cleanup: delete old node
	log.Infof("Deleting old node %s...", old.Name)
	if err := NodeDelete(ctx, runtime, old, k3d.NodeDeleteOpts{SkipLBUpdate: true}); err != nil {
		return err
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
		return nil, err
	}

	result := targetCopy.(*k3d.Node)

	if !opts.keepState {
		// ensure that node state is empty
		result.State = k3d.NodeState{}
	}

	return result, err
}
