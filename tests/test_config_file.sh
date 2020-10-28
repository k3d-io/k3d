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


clustername="ConfigTest"

highlight "[START] ConfigTest $EXTRA_TITLE"

info "Creating cluster $clustername..."
$EXE cluster create "$clustername" --config "$CURR_DIR/assets/config_test_simple.yaml" $EXTRA_FLAG || failed "could not create cluster $clustername $EXTRA_TITLE"

info "Sleeping for 5 seconds to give the cluster enough time to get ready..."
sleep 5

# 1. check initial access to the cluster
info "Checking that we have access to the cluster..."
check_clusters "$clustername" || failed "error checking cluster"

info "Checking that we have 3 nodes online..."
check_multi_node "$clustername" 3 || failed "failed to verify number of nodes"

# 2. TODO: check some config settings


# Cleanup

info "Deleting cluster $clustername..."
$EXE cluster delete "$clustername" || failed "could not delete the cluster $clustername"

highlight "[DONE] ConfigTest $EXTRA_TITLE"

exit 0


