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


: "${EXTRA_FLAG:=""}"
: "${EXTRA_TITLE:=""}"

if [[ -n "$K3S_IMAGE" ]]; then
  EXTRA_FLAG="--image rancher/k3s:$K3S_IMAGE"
  EXTRA_TITLE="(rancher/k3s:$K3S_IMAGE)"
fi

export CURRENT_STAGE="Test | config-file-stdin | $K3S_IMAGE"

configfileoriginal="$CURR_DIR/assets/config_test_simple.yaml"
configfile="/tmp/config_test_simple-tmp_$(date -u +'%Y%m%dT%H%M%SZ').yaml"
clustername="configteststdin"

sed -E "s/^  name:.+/  name: $clustername/g" < "$configfileoriginal" > "$configfile" # replace cluster name in config file so we can use it in this script without running into override issues
cat "$configfile"
highlight "[START] ConfigTest $EXTRA_TITLE"

info "Creating cluster $clustername..."

cat <<EOF | $EXE cluster create "$clustername" --config=-
apiVersion: k3d.io/v1alpha4
kind: Simple
metadata:
  name: test
servers: 3
agents: 2
#image: rancher/k3s:latest
volumes:
  - volume: $HOME:/some/path
    nodeFilters:
      - all
env:
  - envVar: bar=baz,bob
    nodeFilters:
      - all
registries:
  create:
    name: stdintest.registry.localhost
  use: []
  config: |
    mirrors:
      "my.company.registry":
        endpoint:
          - http://my.company.registry:5000
options:
  k3d:
    wait: true
    timeout: "360s" # should be pretty high for multi-server clusters to allow for a proper startup routine
    disableLoadbalancer: false
    disableImageVolume: false
  k3s:
    extraArgs:
      - arg: --tls-san=127.0.0.1
        nodeFilters:
          - server:*
    nodeLabels:
      - label: foo=bar
        nodeFilters:
          - server:0
          - loadbalancer
  kubeconfig:
    updateDefaultKubeconfig: true
    switchCurrentContext: true
  runtime:
    labels:
      - label: foo=bar
        nodeFilters:
          - server:0
          - loadbalancer
EOF

test $? -eq 0 || failed "could not create cluster $clustername $EXTRA_TITLE"

info "Sleeping for 5 seconds to give the cluster enough time to get ready..."
sleep 5

# 1. check initial access to the cluster
info "Checking that we have access to the cluster..."
check_clusters "$clustername" || failed "error checking cluster"

info "Checking that we have 5 nodes online..."
check_multi_node "$clustername" 5 || failed "failed to verify number of nodes"

# 2. check some config settings

## Environment Variables
info "Ensuring that environment variables are present in the node containers as set in the config (with comma)"
exec_in_node "k3d-$clustername-server-0" "env" | grep -q "bar=baz,bob" || failed "Expected env var 'bar=baz,bob' is not present in node k3d-$clustername-server-0"

## Container Labels
info "Ensuring that container labels have been set as stated in the config"
docker_assert_container_label "k3d-$clustername-server-0" "foo=bar" || failed "Expected label 'foo=bar' not present on container/node k3d-$clustername-server-0"

## K3s Node Labels
info "Ensuring that k3s node labels have been set as stated in the config"
k3s_assert_node_label "k3d-$clustername-server-0" "foo=bar" || failed "Expected label 'foo=bar' not present on node k3d-$clustername-server-0"

## Registry Node
registryname="stdintest.registry.localhost"
info "Ensuring, that we have a registry node present"
$EXE node list "$registryname" || failed "Expected registry node $registryname to be present"

## merged registries.yaml
info "Ensuring, that the registries.yaml file contains both registries"
exec_in_node "k3d-$clustername-server-0" "cat /etc/rancher/k3s/registries.yaml" | grep -qi "my.company.registry"  || failed "Expected 'my.company.registry' to be in the /etc/rancher/k3s/registries.yaml"
exec_in_node "k3d-$clustername-server-0" "cat /etc/rancher/k3s/registries.yaml" | grep -qi "$registryname" || failed "Expected '$registryname' to be in the /etc/rancher/k3s/registries.yaml"

# Cleanup

info "Deleting cluster $clustername (using config file)..."
$EXE cluster delete --config "$configfile" --trace || failed "could not delete the cluster $clustername"

rm "$configfile"

highlight "[DONE] ConfigTest $EXTRA_TITLE"

exit 0


