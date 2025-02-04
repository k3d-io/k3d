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
  EXTRA_FLAG="--image rancher/k3s:$K3S_IMAGE"
  EXTRA_TITLE="(rancher/k3s:$K3S_IMAGE)"
fi

export CURRENT_STAGE="Test | registry | $K3S_IMAGE"


clustername="registrytest"
registryname="$clustername-registry"

highlight "[START] RegistryTest $EXTRA_TITLE"

info "Creating cluster $clustername..."
$EXE cluster create "$clustername" --agents 1 --wait --timeout 360s --registry-create "$registryname" $EXTRA_FLAG || failed "could not create cluster $clustername $EXTRA_TITLE"

info "Sleeping for 5 seconds to give the cluster enough time to get ready..."
sleep 5

# 1. check initial access to the cluster
info "Checking that we have access to the cluster..."
check_clusters "$clustername" || failed "error checking cluster"

info "Checking that we have 2 nodes online..."
check_multi_node "$clustername" 2 || failed "failed to verify number of nodes"

# 2. Check that we can discover the LocalRegistryHosting Configmap that points to our registry
info "Checking that we can discover the LocalRegistryHosting Configmap..."
kubectl get configmap -n kube-public local-registry-hosting -o go-template='{{index .data "localRegistryHosting.v1"}}' | grep -q 'host' || failed "failed to discover LocalRegistryHosting Configmap"

# 3. load an image into the registry
info "Pushing an image to the registry..."
registryPort=$(docker inspect $registryname | jq '.[0].NetworkSettings.Ports | with_entries(select(.value | . != null)) | to_entries[0].value[0].HostPort' | sed -E 's/"//g')
docker pull alpine:latest > /dev/null
docker tag alpine:latest "localhost:$registryPort/alpine:local" > /dev/null
docker push "localhost:$registryPort/alpine:local" || failed "Failed to push image to managed registry"

# 4. use imported image
info "Spawning a pod using the pushed image..."
kubectl run --image "$registryname:$registryPort/alpine:local" testimage --command -- tail -f /dev/null
info "Waiting for a bit for the pod to start..."
sleep 5

wait_for_pod_running_by_name "testimage"
wait_for_pod_running_by_label "k8s-app=kube-dns" "kube-system"

sleep 5
# Cleanup

info "Deleting cluster $clustername..."
$EXE cluster delete "$clustername" || failed "could not delete the cluster $clustername"

highlight "[DONE] RegistryTest $EXTRA_TITLE"

exit 0


