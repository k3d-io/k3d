#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

LOG_FILE="$TEST_OUTPUT_DIR/$( basename "${BASH_SOURCE[0]}" ).log"
exec >${LOG_FILE} 2>&1


: "${EXTRA_FLAG:=""}"
: "${EXTRA_TITLE:=""}"

if [[ -n "$K3S_IMAGE_TAG" ]]; then
  EXTRA_FLAG="--image rancher/k3s:$K3S_IMAGE_TAG"
  EXTRA_TITLE="(rancher/k3s:$K3S_IMAGE_TAG)"
fi

export CURRENT_STAGE="Test | config-file | $K3S_IMAGE_TAG"

configfileoriginal="$CURR_DIR/assets/config_test_simple.yaml"
configfile="/tmp/config_test_simple-tmp_$(date -u +'%Y%m%dT%H%M%SZ').yaml"
clustername="configtest"

sed -E "s/^name:.+/name: $clustername/g" < "$configfileoriginal" > "$configfile" # replace cluster name in config file so we can use it in this script without running into override issues

highlight "[START] ConfigTest $EXTRA_TITLE"

info "Creating cluster $clustername..."
$EXE cluster create "$clustername" --config "$configfile" $EXTRA_FLAG || failed_with_logfile "could not create cluster $clustername $EXTRA_TITLE" "$LOG_FILE"

info "Sleeping for 5 seconds to give the cluster enough time to get ready..."
sleep 5

# 1. check initial access to the cluster
info "Checking that we have access to the cluster..."
check_clusters "$clustername" || failed_with_logfile "error checking cluster" "$LOG_FILE"

info "Checking that we have 5 nodes online..."
check_multi_node "$clustername" 5 || failed_with_logfile "failed to verify number of nodes" "$LOG_FILE"

# 2. check some config settings

## Environment Variables
info "Ensuring that environment variables are present in the node containers as set in the config (with comma)"
exec_in_node "k3d-$clustername-server-0" "env" | grep -q "bar=baz,bob" || failed_with_logfile "Expected env var 'bar=baz,bob' is not present in node k3d-$clustername-server-0" "$LOG_FILE"

## Container Labels
info "Ensuring that container labels have been set as stated in the config"
docker_assert_container_label "k3d-$clustername-server-0" "foo=bar" || failed_with_logfile "Expected label 'foo=bar' not present on container/node k3d-$clustername-server-0" "$LOG_FILE"

## K3s Node Labels
info "Ensuring that k3s node labels have been set as stated in the config"
k3s_assert_node_label "k3d-$clustername-server-0" "foo=bar" || failed_with_logfile "Expected label 'foo=bar' not present on node k3d-$clustername-server-0" "$LOG_FILE"

## Registry Node
registryname="registry.localhost"
info "Ensuring, that we have a registry node present"
$EXE node list "$registryname" || failed_with_logfile "Expected registry node $registryname to be present" "$LOG_FILE"

## merged registries.yaml
info "Ensuring, that the registries.yaml file contains both registries"
exec_in_node "k3d-$clustername-server-0" "cat /etc/rancher/k3s/registries.yaml" | grep -qi "my.company.registry"  || failed_with_logfile "Expected 'my.company.registry' to be in the /etc/rancher/k3s/registries.yaml" "$LOG_FILE"
exec_in_node "k3d-$clustername-server-0" "cat /etc/rancher/k3s/registries.yaml" | grep -qi "$registryname" || failed_with_logfile "Expected '$registryname' to be in the /etc/rancher/k3s/registries.yaml" "$LOG_FILE"

# Cleanup

info "Deleting cluster $clustername (using config file)..."
$EXE cluster delete --config "$configfile" || failed_with_logfile "could not delete the cluster $clustername" "$LOG_FILE"

rm "$configfile"

highlight "[DONE] ConfigTest $EXTRA_TITLE"

exit 0


