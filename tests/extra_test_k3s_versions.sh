#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

K3S_VERSIONS=("v1.20.12-k3s1" "v1.21.6-k3s1" "v1.22.3-k3s1")
FAILED_TESTS=()

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

for version in "${K3S_VERSIONS[@]}"; do

  ### Step Setup ###
  # Redirect all stdout/stderr output to logfile
  LOG_FILE="$TEST_OUTPUT_DIR/$( basename "${BASH_SOURCE[0]}" )_${version//./-}.log"
  exec >${LOG_FILE} 2>&1
  export LOG_FILE

  # use a kubeconfig file specific to this test
  KUBECONFIG="$KUBECONFIG_ROOT/$( basename "${BASH_SOURCE[0]}" )_${version//./-}.yaml"
  export KUBECONFIG
  ### Step Setup ###

  export CURRENT_STAGE="Suite | k3s-versions | $version"

  clustername="k3s-version-${version//./-}"


  info "Creating a cluster with k3s version $version ..."
  $EXE cluster create $clustername --wait --timeout 60s --image "rancher/k3s:$version" || failed "could not create cluster with k3s version $version"

  info "Checking we have access to the cluster ..."
  check_clusters "$clustername" || failed "error checking cluster with k3s version $version"

  info "Deleting cluster ..."
  $EXE cluster delete $clustername || failed "could not delete the cluster $clustername"

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
  exit 1
else
  passed "Successfully verified all given k3s versions"
  exit 0
fi
