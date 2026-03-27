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

export CURRENT_STAGE="Test | registry-use | $K3S_IMAGE"

highlight "[START] RegistryUseTest $EXTRA_TITLE"

#########################################################################
# Scenario 1: Exact reproduction of issue #1642
#   k3d registry create mycluster-registry  → container: k3d-mycluster-registry
#   k3d cluster create --registry-use mycluster-registry mycluster
# The auto-generated create name "k3d-mycluster-registry" collides with
# the existing registry. --registry-use must NOT trigger creation.
#########################################################################

section "Scenario 1: Issue #1642 exact reproduction (name collision)"

clustername_1="regusetest1"
registryname_1="${clustername_1}-registry"
registryfull_1="k3d-${registryname_1}"

info "Creating standalone registry (short name: $registryname_1, container: $registryfull_1)..."
$EXE registry create "$registryname_1" $EXTRA_FLAG || failed "could not create registry $registryfull_1"

docker inspect "$registryfull_1" > /dev/null 2>&1 || failed "registry container $registryfull_1 not found"

# Pass the SHORT name (without k3d- prefix), matching the issue's exact steps.
# The cluster name equals the registry short name minus "-registry", so the
# auto-generated create name "k3d-<cluster>-registry" collides.
info "Creating cluster $clustername_1 with --registry-use $registryname_1 (short name)..."
$EXE cluster create "$clustername_1" --agents 1 --wait --timeout 360s --registry-use "$registryname_1" $EXTRA_FLAG || failed "could not create cluster $clustername_1 with --registry-use $registryname_1"

info "Sleeping for 5 seconds to give the cluster enough time to get ready..."
sleep 5

info "Checking that we have access to the cluster..."
check_clusters "$clustername_1" || failed "error checking cluster $clustername_1"

info "Checking that we have 2 nodes online..."
check_multi_node "$clustername_1" 2 || failed "failed to verify number of nodes for $clustername_1"

# Verify the registry is functional
registryPort_1=$(docker inspect "$registryfull_1" | jq '.[0].NetworkSettings.Ports["5000/tcp"][0].HostPort' | sed -E 's/"//g')
check_url "http://localhost:$registryPort_1/v2/_catalog" || failed "registry $registryfull_1 not accessible"

info "Cleaning up scenario 1..."
$EXE cluster delete "$clustername_1" || failed "could not delete cluster $clustername_1"
$EXE registry delete "$registryfull_1" || failed "could not delete registry $registryfull_1"

passed "Scenario 1 passed"

#########################################################################
# Scenario 2: --registry-use with full k3d-prefixed name
#   Ensures the fix works when using the full container name too.
#########################################################################

section "Scenario 2: --registry-use with full k3d-prefixed name"

clustername_2="regusetest2"
registryname_2="${clustername_2}-registry"
registryfull_2="k3d-${registryname_2}"

info "Creating standalone registry $registryfull_2..."
$EXE registry create "$registryname_2" $EXTRA_FLAG || failed "could not create registry $registryfull_2"

docker inspect "$registryfull_2" > /dev/null 2>&1 || failed "registry container $registryfull_2 not found"

# Pass the FULL name (with k3d- prefix)
info "Creating cluster $clustername_2 with --registry-use $registryfull_2 (full name)..."
$EXE cluster create "$clustername_2" --agents 1 --wait --timeout 360s --registry-use "$registryfull_2" $EXTRA_FLAG || failed "could not create cluster $clustername_2 with --registry-use $registryfull_2"

info "Sleeping for 5 seconds to give the cluster enough time to get ready..."
sleep 5

info "Checking that we have access to the cluster..."
check_clusters "$clustername_2" || failed "error checking cluster $clustername_2"

info "Checking that we have 2 nodes online..."
check_multi_node "$clustername_2" 2 || failed "failed to verify number of nodes for $clustername_2"

# Verify the registry is functional: push and pull an image
registryPort_2=$(docker inspect "$registryfull_2" | jq '.[0].NetworkSettings.Ports["5000/tcp"][0].HostPort' | sed -E 's/"//g')
info "Pushing an image to the registry..."
docker pull alpine:latest > /dev/null
docker tag alpine:latest "localhost:$registryPort_2/alpine:local" > /dev/null
docker push "localhost:$registryPort_2/alpine:local" || failed "Failed to push image to registry"

info "Spawning a pod using the pushed image..."
kubectl run --image "$registryfull_2:$registryPort_2/alpine:local" testimage --command -- tail -f /dev/null
info "Waiting for the pod to start..."
sleep 5

wait_for_pod_running_by_name "testimage"

info "Cleaning up scenario 2..."
$EXE cluster delete "$clustername_2" || failed "could not delete cluster $clustername_2"
$EXE registry delete "$registryfull_2" || failed "could not delete registry $registryfull_2"

passed "Scenario 2 passed"

highlight "[DONE] RegistryUseTest $EXTRA_TITLE"

exit 0
