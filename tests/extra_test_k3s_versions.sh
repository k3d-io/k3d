#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

K3S_VERSIONS=("v1.17.17-k3s1" "v1.18.15-k3s1" "v1.19.7-k3s1" "v1.20.2-k3s1")
FAILED_TESTS=()

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

for version in "${K3S_VERSIONS[@]}"; do

  export CURRENT_STAGE="Suite | k3s-versions | $version"

  info "Creating a cluster with k3s version $version ..."
  $EXE cluster create c1 --wait --timeout 60s --image "rancher/k3s:$version" || failed "could not create cluster with k3s version $version"

  info "Checking we have access to the cluster ..."
  check_clusters "c1" || failed "error checking cluster with k3s version $version"

  info "Deleting cluster ..."
  $EXE cluster delete c1 || failed "could not delete the cluster c1"

  K3S_IMAGE_TAG="$version" $CURR_DIR/test_full_lifecycle.sh
  if [[ $? -eq 1 ]]; then
    FAILED_TESTS+=("full_lifecycle: $version")
  fi

  $EXE cluster rm -a || failed "failed to delete clusters"

  K3S_IMAGE_TAG="$version" $CURR_DIR/test_multi_master.sh
  if [[ $? -eq 1 ]]; then
    FAILED_TESTS+=("multi_master: $version")
  fi

  $EXE cluster rm -a || failed "failed to delete clusters"

  K3S_IMAGE_TAG="$version" $CURR_DIR/test_multi_master_start_stop.sh
  if [[ $? -eq 1 ]]; then
    FAILED_TESTS+=("multi_master_start_stop: $version")
  fi

  $EXE cluster rm -a || failed "failed to delete clusters"

done

if [[ ${#FAILED_TESTS[@]} -gt 0 ]]; then
  warn "FAILED TESTS"
  for failed_test in "${FAILED_TESTS[@]}"; do
    warn "- $failed_test"
  done
else
  passed "Successfully verified all given k3s versions"
  exit 0
fi
