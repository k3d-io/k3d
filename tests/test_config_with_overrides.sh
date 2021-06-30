#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"


: "${EXTRA_FLAG:=""}"
: "${EXTRA_TITLE:=""}"

if [[ -n "$K3S_IMAGE_TAG" ]]; then
  EXTRA_FLAG="--image rancher/k3s:$K3S_IMAGE_TAG"
  EXTRA_TITLE="(rancher/k3s:$K3S_IMAGE_TAG)"
fi


clustername="cfgoverridetest"

highlight "[START] Config With Override $EXTRA_TITLE"

info "Creating cluster $clustername..."
$EXE cluster create "$clustername" --config "$CURR_DIR/assets/config_test_simple.yaml" --servers 4 -v /tmp/test:/tmp/test@loadbalancer --registry-create=false --env "x=y@agent:1" $EXTRA_FLAG  || failed "could not create cluster $clustername $EXTRA_TITLE"

info "Sleeping for 5 seconds to give the cluster enough time to get ready..."
sleep 5

# 1. check initial access to the cluster
info "Checking that we have access to the cluster..."
check_clusters "$clustername" || failed "error checking cluster"

info "Checking that we have 6 nodes online..."
check_multi_node "$clustername" 6 || failed "failed to verify number of nodes"

# 2. check some config settings

## Environment Variables
info "Ensuring that environment variables are present in the node containers as set in the config and overrides"
exec_in_node "k3d-$clustername-server-0" "env" | grep "bar=baz" || failed "Expected env var 'bar=baz' is not present in node k3d-$clustername-server-0"
exec_in_node "k3d-$clustername-agent-1" "env" | grep "x=y" || failed "Expected env var 'x=y' is not present in node k3d-$clustername-agent-1"

## Container Labels
info "Ensuring that container labels have been set as stated in the config"
docker_assert_container_label "k3d-$clustername-server-0" "foo=bar" || failed "Expected label 'foo=bar' not present on container/node k3d-$clustername-server-0"

## K3s Node Labels
info "Ensuring that k3s node labels have been set as stated in the config"
k3s_assert_node_label "k3d-$clustername-server-0" "foo=bar" || failed "Expected label 'foo=bar' not present on node k3d-$clustername-server-0"

## Registry Node
info "Ensuring, that we DO NOT have a registry node present"
$EXE node list "k3d-$clustername-registry" && failed "Expected k3d-$clustername-registry to NOT be present"

## merged registries.yaml
info "Ensuring, that the registries.yaml file contains both registries"
exec_in_node "k3d-$clustername-server-0" "cat /etc/rancher/k3s/registries.yaml" | grep -i "my.company.registry"  || failed "Expected 'my.company.registry' to be in the /etc/rancher/k3s/registries.yaml"
exec_in_node "k3d-$clustername-server-0" "cat /etc/rancher/k3s/registries.yaml" | grep -i "k3d-$clustername-registry" && failed "Expected 'k3d-$clustername-registry' to NOT be in the /etc/rancher/k3s/registries.yaml"

# Cleanup

info "Deleting cluster $clustername..."
$EXE cluster delete "$clustername" || failed "could not delete the cluster $clustername"

highlight "[DONE] ConfigTest $EXTRA_TITLE"

exit 0


