#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

KNOWN_TO_FAIL=("v1.17.17-k3s1" "v1.18.15-k3s1") # some versions of k3s don't work here (dqlite), so we can skip them here

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

: "${EXTRA_FLAG:=""}"
: "${EXTRA_TITLE:=""}"


export CURRENT_STAGE="Test | multi-server-start-stop | $K3S_IMAGE_TAG"

if [[ -n "$K3S_IMAGE_TAG" ]]; then
  for failing in "${KNOWN_TO_FAIL[@]}"; do
    if [[ "$failing" == "$K3S_IMAGE_TAG" ]]; then
      warn "$K3S_IMAGE_TAG is known to fail this test. Skipping."
      exit 0
    fi
  done
  EXTRA_FLAG="--image rancher/k3s:$K3S_IMAGE_TAG"
  EXTRA_TITLE="(rancher/k3s:$K3S_IMAGE_TAG)"
fi


info "Creating cluster multiserver $EXTRA_TITLE ..."
$EXE cluster create "multiserver" --servers 3 --api-port 6443 --wait --timeout 360s $EXTRA_FLAG || failed "could not create cluster multiserver $EXTRA_TITLE"
info "Checking that we have access to the cluster..."
check_clusters "multiserver" || failed "error checking cluster $EXTRA_TITLE"

info "Sleeping for 5 seconds to give the cluster enough time to get ready..."
sleep 5

info "Checking that we have 3 server nodes online..."
check_multi_node "multiserver" 3 || failed "failed to verify number of nodes $EXTRA_TITLE"

info "Stopping cluster..."
$EXE cluster stop "multiserver" || failed "failed to stop cluster"

info "Waiting for a bit..."
sleep 5

info "Restarting cluster (time: $(date -u +"%Y-%m-%d %H:%M:%S %Z"))..."
$EXE cluster start multiserver --timeout 360s || failed "failed to restart cluster (timeout 360s)"

info "Sleeping for 5 seconds to give the cluster enough time to get ready..."
sleep 5

info "Checking that we have access to the cluster..."
check_clusters "multiserver" || failed "failed to verify that we have access to the cluster"

info "Deleting cluster multiserver..."
$EXE cluster delete "multiserver" || failed "could not delete the cluster multiserver $EXTRA_TITLE"

passed "GOOD: multiserver cluster test successful $EXTRA_TITLE"

exit 0


