#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

LOG_FILE="$TEST_OUTPUT_DIR/$( basename "${BASH_SOURCE[0]}" ).log"
exec >${LOG_FILE} 2>&1

: "${EXTRA_FLAG:=""}"
: "${EXTRA_TITLE:=""}"

if [[ -n "$K3S_IMAGE_TAG" ]]; then
  EXTRA_FLAG="--image rancher/k3s:$K3S_IMAGE_TAG"
  EXTRA_TITLE="(rancher/k3s:$K3S_IMAGE_TAG)"
fi

export CURRENT_STAGE="Test | multi-server | $K3S_IMAGE_TAG"

clustername="multiserver"

info "Creating cluster $clustername $EXTRA_TITLE ..."
$EXE cluster create "$clustername" --servers 3 --api-port 6443 --wait --timeout 360s $EXTRA_FLAG || failed "could not create cluster $clustername $EXTRA_TITLE"
info "Checking that we have access to the cluster..."
check_clusters "$clustername" || failed "error checking cluster $EXTRA_TITLE"

info "Sleeping for 5 seconds to give the cluster enough time to get ready..."
sleep 5

info "Checking that we have 3 server nodes online..."
check_multi_node "$clustername" 3 || failed "failed to verify number of nodes $EXTRA_TITLE"

info "Deleting cluster $clustername..."
$EXE cluster delete "$clustername" || failed "could not delete the cluster $clustername $EXTRA_TITLE"

passed "GOOD: $clustername cluster test successful $EXTRA_TITLE"

exit 0


