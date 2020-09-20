#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

K3S_VERSIONS=("v1.17.12-k3s1" "v1.18.9-k3s1" "v1.19.2-rc2-k3s1" "v1.19.1-k3s1")
FAILED_VERSIONS=()

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

for version in "${K3S_VERSIONS[@]}"; do

  info "Creating a cluster with k3s version $version ..."
  $EXE cluster create c1 --wait --timeout 60s --api-port 6443 --image "rancher/k3s:$version" || failed "could not create cluster with k3s version $version"

  info "Checking we have access to the cluster ..."
  check_clusters "c1" || failed "error checking cluster with k3s version $version"

  info "Deleting cluster ..."
  $EXE cluster delete c1 || failed "could not delete the cluster c1"

  K3S_IMAGE_TAG="$version" $CURR_DIR/test_multi_master.sh
  if [[ $? -eq 1 ]]; then
    FAILED_VERSIONS+=("$version")
  fi

done

if [[ ${#FAILED_VERSIONS[@]} -gt 0 ]]; then
  failed "Tests failed for k3s versions: ${FAILED_VERSIONS[*]}"
else
  passed "Successfully verified all given k3s versions"
  exit 0
fi
