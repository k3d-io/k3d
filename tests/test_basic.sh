#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

info "Creating two clusters..."
$EXE create cluster c1 --wait --timeout 60s --api-port 6443 || failed "could not create cluster c1"
$EXE create cluster c2 --wait --timeout 60s --api-port 6444 || failed "could not create cluster c2"

info "Checking that we can get both clusters..."
check_cluster_count 2

info "Checking we have access to both clusters..."
check_clusters "c1" "c2" || failed "error checking cluster"

info "Check k3s token retrieval"
check_cluster_token_exist "c1" || failed "could not find cluster token c1"
check_cluster_token_exist "c2" || failed "could not find cluster token c2"

info "Deleting clusters..."
$EXE delete cluster c1 || failed "could not delete the cluster c1"
$EXE delete cluster c2 || failed "could not delete the cluster c2"

exit 0


