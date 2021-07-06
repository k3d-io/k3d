#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

export CURRENT_STAGE="Test | Loadbalancer"

highlight "[START] LoadbalancerTest $EXTRA_TITLE"

function check_container_port() {
  # $1 = container name
  # $2 = wanted port
  exists=$(docker inspect "$1" --format '{{ range $k, $_ := .NetworkSettings.Ports }}{{ if eq $k "'"$2"'" }}true{{ end }}{{ end }}')
  if [[ $exists == "true" ]]; then
    return 0
  else
    docker inspect "$1" --format '{{ range $k, $_ := .NetworkSettings.Ports }}{{ printf "%s\n" $k }}{{ end }}'
    return 1
  fi
}

clustername="lbtest"

info "Creating cluster $clustername..."
$EXE cluster create $clustername --timeout 360s --agents 1 -p 2222:3333@server:0 -p 8080:80@server:0:proxy -p 1234:4321/tcp@agent:0:direct || failed "could not create cluster $clustername"

info "Checking we have access to the cluster..."
check_clusters "$clustername" || failed "error checking cluster"

info "Checking Container Ports..."

info "> Checking automatic port mapping for Kube API on loadbalancer (6443)..."
check_container_port k3d-$clustername-serverlb "6443/tcp" || failed "6443/tcp not on serverlb"

info "> Checking explicit proxy port mapping of port 80 -> loadbalancer -> server-0"
check_container_port k3d-$clustername-serverlb "80/tcp" || failed "80/tcp not on serverlb"

info "> Checking explicit direct port mapping of port 4321 -> agent-0"
check_container_port k3d-$clustername-agent-0 "4321/tcp" || failed "4321/tcp not on agent-0"

info "> Checking implicit proxy port mapping of port 3333 -> loadbalancer -> server-0"
check_container_port k3d-$clustername-server-0 "3333/tcp" && failed "3333/tcp on server-0 but should be on serverlb"
check_container_port k3d-$clustername-serverlb "3333/tcp" || failed "3333/tcp not on serverlb"

info "Checking Loadbalancer Config..."
$EXE debug loadbalancer get-config $clustername | grep -A1 "80.tcp" | grep "k3d-$clustername-server-0" || failed "port 80.tcp not configured for server-0"

info "Deleting clusters..."
$EXE cluster delete $clustername || failed "could not delete the cluster $clustername"

exit 0


