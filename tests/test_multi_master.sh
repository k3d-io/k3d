#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

info "Creating cluster multimaster..."
$EXE --verbose create cluster "multimaster" --masters 3 --api-port 6443 --wait --timeout 360s || failed "could not create cluster multimaster"

info "Checking we have access to the cluster..."
check_k3d_clusters "multimaster" || failed "error checking cluster"

info "Checking that we have 3 servers online..."
check_multi_master() {
  for c in "$@" ; do
    kc=$($EXE get kubeconfig "$c")
    [ -n "$kc" ] || abort "could not obtain a kubeconfig for $c"
    nodeCount=$(kubectl --kubeconfig="$kc" get nodes -o=custom-columns=NAME:.metadata.name --no-headers | wc -l)
    if [[ $nodeCount == 3 ]]; then
      passed "cluster $c has 3 nodes, as expected"
    else
      warn "cluster $c has incorrect number of nodes: $nodeCount != 3"
      kubectl --kubeconfig="$kc" get nodes -o=custom-columns=NAME:.metadata.name --no-headers
      return 1
    fi
  done
  return 0
}
check_multi_master "multimaster"

info "Deleting cluster multimaster..."
$EXE delete cluster "multimaster" || failed "could not delete the cluster multimaster"

exit 0


