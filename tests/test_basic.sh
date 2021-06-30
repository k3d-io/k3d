#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

export CURRENT_STAGE="Test | basic"

info "Creating two clusters..."
$EXE cluster create c1 --wait --timeout 60s --api-port 6443 --env 'TEST_VAR=user\@pass\\@server:0' || failed "could not create cluster c1"
$EXE cluster create c2 --wait --timeout 60s || failed "could not create cluster c2"

info "Checking that we can get both clusters..."
check_cluster_count 2

info "Checking we have access to both clusters..."
check_clusters "c1" "c2" || failed "error checking cluster"

info "Checking cluster env var with escaped @ signs..."
docker exec k3d-c1-server-0 env | grep -E '^TEST_VAR=user@pass\\$' || failed "Failed to lookup proper env var in container"

info "Check k3s token retrieval"
check_cluster_token_exist "c1" || failed "could not find cluster token c1"
check_cluster_token_exist "c2" || failed "could not find cluster token c2"

info "Deleting clusters..."
$EXE cluster delete c1 || failed "could not delete the cluster c1"
$EXE cluster delete c2 || failed "could not delete the cluster c2"

exit 0


