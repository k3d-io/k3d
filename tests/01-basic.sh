#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

#########################################################################################

FIXTURES_DIR=$CURR_DIR/fixtures

# a dummy registries.yaml file
REGISTRIES_YAML=$FIXTURES_DIR/01-registries-empty.yaml

#########################################################################################

info "Creating two clusters..."
$EXE create --wait 60 --name "c1" --api-port 6443 -v $REGISTRIES_YAML:/etc/rancher/k3s/registries.yaml || failed "could not create cluster c1"
$EXE create --wait 60 --name "c2" --api-port 6444 || failed "could not create cluster c2"

info "Checking we have access to both clusters..."
check_k3d_clusters "c1" "c2" || failed "error checking cluster"

info "Deleting clusters..."
$EXE delete --name "c1" || failed "could not delete the cluster c1"
$EXE delete --name "c2" || failed "could not delete the cluster c2"

exit 0


