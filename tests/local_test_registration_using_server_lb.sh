#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

export CURRENT_STAGE="local | registration_using_server_lb"

clean_up() {
  info "Cleaning up"
  $BIN cluster delete $CLUSTER_NAME > /dev/null || true
}

check_k3s_url() {
  node=$1
  expected_url=$2
  curr_url=$(docker inspect $node | grep K3S_URL | awk -F"=" '{print $2}' | sed 's/",//')

  if [ "$curr_url" != "$expected_url" ]; then
    failed "Default K3S URL mismatch expected: '${expected_url}' got: ${curr_url}"
  else
    passed "$curr_url matches the expected: '${expected_url}'"
  fi
}

run() {
  cmd=$1
  eval "${cmd} > /dev/null"
}

# Constants for tests
BIN=~/code/k3d/bin/k3d
CLUSTER_NAME=test-lb-registration
K3S_URL_DEFAULT="https://k3d-${CLUSTER_NAME}-server-0:6443"
K3S_URL_LB="https://k3d-${CLUSTER_NAME}-serverlb:6443"

clean_up
info "Starting a multi server cluster"
run "$BIN cluster create $CLUSTER_NAME -s 3"

info "Adding an agent and checking its K3S_URL"
run "$BIN node create -c $CLUSTER_NAME testagent"
check_k3s_url k3d-testagent-0 $K3S_URL_DEFAULT
run "$BIN node delete k3d-testagent-0"

info "Deleting server-0"
run "$BIN node delete k3d-${CLUSTER_NAME}-server-0"
run "kubectl delete node k3d-${CLUSTER_NAME}-server-0"

info "Adding a new agent to check the new K3S_URL"
run "$BIN node create -c $CLUSTER_NAME testagent1"
check_k3s_url k3d-testagent1-0 $K3S_URL_LB
run "$BIN node delete k3d-testagent1-0"

info "Adding a new server to check K3S_URL"
run "$BIN node create -c $CLUSTER_NAME testserver --role server"
check_k3s_url k3d-testserver-0 $K3S_URL_LB
run "$BIN node delete k3d-testserver-0"

clean_up

info "Adding a cluster with no lb to check K3S_URL"
run "$BIN cluster create $CLUSTER_NAME -s 3 --no-lb"

run "$BIN node create -c $CLUSTER_NAME testagent1"
check_k3s_url k3d-testagent1-0 $K3S_URL_DEFAULT
run "$BIN node delete k3d-testagent1-0"

clean_up
