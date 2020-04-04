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

info "Creating two clusters c1 and c2..."
$EXE create --name "c1" --api-port 6443 -v $REGISTRIES_YAML:/etc/rancher/k3s/registries.yaml || failed "could not create cluster c1"
$EXE create --name "c2" --api-port 6444 || failed "could not create cluster c2"

info "Checking we have access to both clusters..."
check_k3d_clusters "c1" "c2" || failed "error checking cluster"
dump_registries_yaml_in "c1" "c2"

info "Creating a cluster with a wrong --registries-file argument..."
$EXE create --name "c3" --api-port 6445 --registries-file /etc/inexistant || passed "expected error with a --registries-file that does not exist"

info "Attaching a container to c2"
background=$(docker run -d --rm alpine sleep 3000)
docker network connect "k3d-c2" "$background"

info "Deleting clusters c1 and c2..."
$EXE delete --name "c1"         || failed "could not delete the cluster c1"
$EXE delete --name "c2" --prune || failed "could not delete the cluster c2"

info "Stopping attached container"
docker stop "$background" >/dev/null

exit 0


