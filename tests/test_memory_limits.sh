#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

export CURRENT_STAGE="Test | MemoryLimits"

highlight "[START] MemoryLimitTest $EXTRA_TITLE"

clustername="memlimittest"

info "Creating cluster $clustername..."
$EXE cluster create $clustername --timeout 360s --servers-memory 1g --agents 1 --agents-memory 1.5g || failed "could not create cluster $clustername"

info "Checking we have access to the cluster..."
check_clusters "$clustername" || failed "error checking cluster"

info "Checking Memory Limits (docker)..."
if [[ $(docker inspect k3d-$clustername-server-0 | jq '.[0].HostConfig.Memory') != "1073741824" ]]; then
  failed "Server Memory not set to 1g as expected (docker)"
fi
if [[ $(docker inspect k3d-$clustername-agent-0 | jq '.[0].HostConfig.Memory') != "1610612736" ]]; then
  failed "Agent Memory not set to 1.5g as expected (docker)"
fi

info "Checking Memory Limits (Kubernetes)..."
if [[ $(kubectl get node k3d-$clustername-server-0 -o go-template='{{ .status.capacity.memory }}') != "1073741Ki" ]]; then
  failed "Server Memory not set to 1g as expected (k8s)"
fi
if [[ $(kubectl get node k3d-$clustername-agent-0 -o go-template='{{ .status.capacity.memory }}') != "1610612Ki" ]]; then
  failed "Agent Memory not set to 1.5g as expected (k8s)"
fi

info "Deleting clusters..."
$EXE cluster delete $clustername || failed "could not delete the cluster $clustername"

exit 0


