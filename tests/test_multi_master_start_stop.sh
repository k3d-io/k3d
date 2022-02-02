#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

KNOWN_TO_FAIL=("v1.17.17-k3s1" "v1.18.15-k3s1") # some versions of k3s don't work here (dqlite), so we can skip them here

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


export CURRENT_STAGE="Test | multi-server-start-stop | $K3S_IMAGE"

if [[ -n "$K3S_IMAGE" ]]; then
  for failing in "${KNOWN_TO_FAIL[@]}"; do
    if [[ "$failing" == "$K3S_IMAGE" ]]; then
      warn "$K3S_IMAGE is known to fail this test. Skipping."
      exit 0
    fi
  done
  EXTRA_FLAG="--image rancher/k3s:$K3S_IMAGE"
  EXTRA_TITLE="(rancher/k3s:$K3S_IMAGE)"
fi

clustername="multiserverstartstop"

info "Creating cluster $clustername $EXTRA_TITLE ..."
$EXE cluster create "$clustername" --servers 3 --wait --timeout 360s $EXTRA_FLAG || failed "could not create cluster $clustername $EXTRA_TITLE"
info "Checking that we have access to the cluster..."
check_clusters "$clustername" || failed "error checking cluster $EXTRA_TITLE"

info "Sleeping for 5 seconds to give the cluster enough time to get ready..."
sleep 5

info "Checking that we have 3 server nodes online..."
check_multi_node "$clustername" 3 || failed "failed to verify number of nodes $EXTRA_TITLE"

info "Stopping cluster..."
$EXE cluster stop "$clustername" || failed "failed to stop cluster"

info "Waiting for a bit..."
sleep 5

info "Restarting cluster (time: $(date -u +"%Y-%m-%d %H:%M:%S %Z"))..."
$EXE cluster start $clustername --timeout 360s || failed "failed to restart cluster (timeout 360s)"

info "Sleeping for 5 seconds to give the cluster enough time to get ready..."
sleep 5

info "Checking that we have access to the cluster..."
check_clusters "$clustername" || failed "failed to verify that we have access to the cluster"

info "Deleting cluster $clustername..."
$EXE cluster delete "$clustername" || failed "could not delete the cluster $clustername $EXTRA_TITLE"

passed "GOOD: $clustername cluster test successful $EXTRA_TITLE"

exit 0


