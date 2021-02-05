#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"


: "${EXTRA_FLAG:=""}"
: "${EXTRA_TITLE:=""}"

if [[ -n "$K3S_IMAGE_TAG" ]]; then
  EXTRA_FLAG="--image rancher/k3s:$K3S_IMAGE_TAG"
  EXTRA_TITLE="(rancher/k3s:$K3S_IMAGE_TAG)"
fi

export CURRENT_STAGE="Test | lifecycle | $K3S_IMAGE_TAG"


clustername="lifecycletest"

highlight "[START] Lifecycletest $EXTRA_TITLE"

info "Creating cluster $clustername..."
$EXE cluster create "$clustername" --agents 1 --api-port 6443 --wait --timeout 360s $EXTRA_FLAG || failed "could not create cluster $clustername $EXTRA_TITLE"

info "Sleeping for 5 seconds to give the cluster enough time to get ready..."
sleep 10

# 1. check initial access to the cluster
info "Checking that we have access to the cluster..."
check_clusters "$clustername" || failed "error checking cluster"

info "Checking that we have 2 nodes online..."
check_multi_node "$clustername" 2 || failed "failed to verify number of nodes"

# 2. stop the cluster
info "Stopping cluster..."
$EXE cluster stop "$clustername"

info "Checking that cluster was stopped"
check_clusters "$clustername" && failed "cluster was not stopped, since we still have access"

# 3. start the cluster
info "Starting cluster..."
$EXE cluster start "$clustername" --wait --timeout 360s || failed "cluster didn't come back in time"

info "Checking that we have access to the cluster..."
check_clusters "$clustername" || failed "error checking cluster"

info "Checking that we have 2 nodes online..."
check_multi_node "$clustername" 2 || failed "failed to verify number of nodes"

# 4. adding another agent node
info "Adding one agent node..."
$EXE node create "extra-agent" --cluster "$clustername" --role "agent" --wait --timeout 360s || failed "failed to add agent node"

info "Checking that we have 3 nodes available now..."
check_multi_node "$clustername" 3 || failed "failed to verify number of nodes"

# 4. load an image into the cluster
info "Importing an image into the cluster..."
docker pull alpine:latest > /dev/null
docker tag alpine:latest alpine:local > /dev/null
$EXE image import alpine:local -c $clustername || failed "could not import image in $clustername"

# 5. use imported image
info "Spawning a pod using the imported image..."
kubectl run --image alpine:local testimage --command -- tail -f /dev/null
info "Waiting for a bit for the pod to start..."
sleep 5

wait_for_pod_running_by_name "testimage"
wait_for_pod_running_by_label "k8s-app=kube-dns" "kube-system"

sleep 5

# 6. test host.k3d.internal
info "Checking DNS Lookup for host.k3d.internal"
wait_for_pod_exec "testimage" "nslookup host.k3d.internal" 15 || failed "DNS Lookup for host.k3d.internal failed"

# Cleanup

info "Deleting cluster $clustername..."
$EXE cluster delete "$clustername" || failed "could not delete the cluster $clustername"

highlight "[DONE] Lifecycletest $EXTRA_TITLE"

exit 0


