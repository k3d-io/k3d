#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

info "Creating cluster multiserver..."
$EXE cluster create "multiserver" --servers 3 --api-port 6443 --wait --timeout 360s || failed "could not create cluster multiserver"

info "Checking that we have access to the cluster..."
check_clusters "multiserver" || failed "error checking cluster"

info "Sleeping for 5 seconds to give the cluster enough time to get ready..."
sleep 5

info "Checking that we have 3 server nodes online..."
check_multi_node "multiserver" 3 || failed "failed to verify number of nodes"

info "Deleting cluster multiserver..."
$EXE cluster delete "multiserver" || failed "could not delete the cluster multiserver"

exit 0


