#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

### Step Setup ###
# Redirect all stdout/stderr output to logfile
LOG_FILE="$TEST_OUTPUT_DIR/$( basename "${BASH_SOURCE[0]}" ).log"
exec >${LOG_FILE} 2>&1
export LOG_FILE

# use a kubeconfig file specific to this test
KUBECONFIG="$KUBECONFIG_ROOT/$( basename "${BASH_SOURCE[0]}" ).yaml"
export KUBECONFIG
### Step Setup ###


: "${EXTRA_FLAG:=""}"
: "${EXTRA_TITLE:=""}"

if [[ -n "$K3S_IMAGE" ]]; then
    EXTRA_FLAG="--image $K3S_IMAGE"
    EXTRA_TITLE="($K3S_IMAGE)"
fi

export CURRENT_STAGE="Test | lifecycle | $K3S_IMAGE"


clustername="lifecycletest"

highlight "[START] Lifecycletest $EXTRA_TITLE"

info "Creating cluster $clustername..."
$EXE cluster create "$clustername" --agents 1 --wait --timeout 360s $EXTRA_FLAG || failed "could not create cluster $clustername $EXTRA_TITLE"

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
docker pull iwilltry42/dnsutils:20.04 > /dev/null
docker tag iwilltry42/dnsutils:20.04 testimage:local > /dev/null
$EXE image import testimage:local -c $clustername || failed "could not import image in $clustername"

# 5. use imported image
info "Spawning a pod using the imported image..."
kubectl run --image testimage:local testimage --command -- tail -f /dev/null
info "Waiting for a bit for the pod to start..."
sleep 5

kubectl delete pod -n kube-system -l k8s-app=kube-dns  > /dev/null 2>&1 # delete coredns to force reload of config (reload plugin uses default 30s, which will make tests below fail)
wait_for_pod_running_by_name "testimage"
wait_for_pod_running_by_label "k8s-app=kube-dns" "kube-system"

sleep 5

# 6. test host.k3d.internal
info "Checking DNS Lookup for host.k3d.internal via Ping..."
kubectl describe cm coredns -n kube-system | grep "host.k3d.internal" > /dev/null 2>&1 || failed "Couldn't find host.k3d.internal in CoreDNS configmap"
wait_for_pod_exec "testimage" "ping -c1 host.k3d.internal" 15 || failed "Pinging host.k3d.internal failed"

# Cleanup

info "Deleting cluster $clustername..."
$EXE cluster delete "$clustername" || failed "could not delete the cluster $clustername"

highlight "[DONE] Lifecycletest $EXTRA_TITLE"

exit 0


