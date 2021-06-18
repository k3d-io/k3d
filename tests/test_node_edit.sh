#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

export CURRENT_STAGE="Test | NodeEdit"

highlight "[START] NodeEdit"

clustername="test-node-edit"

existingPortMappingHostPort="1111"
existingPortMappingContainerPort="2222"
newPortMappingHostPort="3333"
newPortMappingContainerPort="4444"

info "Creating cluster $clustername..."
$EXE cluster create $clustername --timeout 360s --port "$existingPortMappingHostPort:$existingPortMappingContainerPort@loadbalancer" || failed "could not create cluster $clustername"

info "Checking cluster access..."
check_clusters "$clustername" || failed "error checking cluster access"

info "Adding port-mapping to loadbalancer..."
$EXE node edit k3d-$clustername-serverlb --port-add $existingPortMappingHostPort:$existingPortMappingContainerPort --port-add $newPortMappingHostPort:$newPortMappingContainerPort || failed "failed to add port-mapping to serverlb in $clustername"

info "Checking port-mappings..."
docker inspect k3d-$clustername-serverlb --format '{{ range $k, $v := .NetworkSettings.Ports }}{{ printf "%s->%s\n" $k $v }}{{ end }}' | grep -E "^$existingPortMappingContainerPort" || failed "failed to verify pre-existing port-mapping"
docker inspect k3d-$clustername-serverlb --format '{{ range $k, $v := .NetworkSettings.Ports }}{{ printf "%s->%s\n" $k $v }}{{ end }}' | grep -E "^$newPortMappingContainerPort" || failed "failed to verify pre-existing port-mapping"

info "Checking cluster access..."
check_clusters "$clustername" || failed "error checking cluster access"

info "Deleting cluster $clustername..."
$EXE cluster delete $clustername || failed "failed to delete the cluster $clustername"

exit 0
