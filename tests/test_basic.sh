#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

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

export CURRENT_STAGE="Test | basic"

clustername_1="test-basic-1"
clustername_2="test-basic-2"

info "Creating two clusters..."
$EXE cluster create $clustername_1 --wait --timeout 60s --env 'TEST_VAR=user\@pass\\@server:0' || failed "could not create cluster $clustername_1"
$EXE cluster create $clustername_2 --wait --timeout 60s || failed "could not create cluster $clustername_2"

info "Checking that we can get both clusters..."
check_cluster_count 2 "$clustername_1" "$clustername_2"

info "Checking we have access to both clusters..."
check_clusters "$clustername_1" "$clustername_2" || failed "error checking cluster"

info "Checking cluster env var with escaped @ signs..."
docker exec k3d-$clustername_1-server-0 env | grep -qE '^TEST_VAR=user@pass\\$' || failed "Failed to lookup proper env var in container"

info "Check k3s token retrieval"
check_cluster_token_exist "$clustername_1" || failed "could not find cluster token $clustername_1"
check_cluster_token_exist "$clustername_2" || failed "could not find cluster token $clustername_2"

info "Deleting clusters..."
$EXE cluster delete $clustername_1 || failed "could not delete the cluster $clustername_1"
$EXE cluster delete $clustername_2 || failed "could not delete the cluster $clustername_2"

exit 0


