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
package types

import (
	"time"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
)

// NodeWaitForLogMessageRestartWarnTime is the time after which to warn about a restarting container
const NodeWaitForLogMessageRestartWarnTime = 2 * time.Minute

var ReadyLogMessagesByRoleAndIntent = map[Role]map[Intent]string{
	Role(InternalRoleInitServer): {
		IntentClusterCreate: "Containerd is now running",
		IntentClusterStart:  "Running kube-apiserver",
		IntentAny:           "Running kube-apiserver",
	},
	ServerRole: {
		IntentAny: "k3s is up and running",
	},
	AgentRole: {
		IntentAny: "Successfully registered node",
	},
	LoadBalancerRole: {
		IntentAny: "start worker processes",
	},
	RegistryRole: {
		IntentAny: "listening on",
	},
}

func GetReadyLogMessage(node *Node, intent Intent) string {
	role := node.Role
	if node.Role == ServerRole && node.ServerOpts.IsInit {
		role = Role(InternalRoleInitServer)
	}
	if _, ok := ReadyLogMessagesByRoleAndIntent[role]; ok {
		if msg, ok := ReadyLogMessagesByRoleAndIntent[role][intent]; ok {
			return msg
		} else {
			if msg, ok := ReadyLogMessagesByRoleAndIntent[role][IntentAny]; ok {
				return msg
			}
		}
	}
	l.Log().Warnf("error looking up ready log message for role %s and intent %s: not defined", role, intent)
	return ""
}
