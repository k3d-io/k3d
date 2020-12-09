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


clustername="registrytest"

highlight "[START] RegistryTest $EXTRA_TITLE"

info "Creating cluster $clustername..."
$EXE cluster create "$clustername" --agents 1 --api-port 6443 --wait --timeout 360s --registry-create $EXTRA_FLAG || failed "could not create cluster $clustername $EXTRA_TITLE"

info "Sleeping for 5 seconds to give the cluster enough time to get ready..."
sleep 5

# 1. check initial access to the cluster
info "Checking that we have access to the cluster..."
check_clusters "$clustername" || failed "error checking cluster"

info "Checking that we have 2 nodes online..."
check_multi_node "$clustername" 2 || failed "failed to verify number of nodes"

# 4. load an image into the cluster
info "Importing an image into the cluster..."
registryPort=$(docker inspect k3d-$clustername-registry | jq '.[0].NetworkSettings.Ports["5000/tcp"][0].HostPort' | sed -E 's/"//g')
docker pull alpine:latest > /dev/null
docker tag alpine:latest k3d-$clustername-registry:$registryPort/alpine:local > /dev/null
docker push k3d-$clustername-registry:$registryPort/alpine:local || fail "Failed to push image to managed registry"

# 5. use imported image
info "Spawning a pod using the imported image..."
kubectl run --image k3d-$clustername-registry:$registryPort/alpine:local testimage --command -- tail -f /dev/null
info "Waiting for a bit for the pod to start..."
sleep 5

wait_for_pod_running_by_name "testimage"
wait_for_pod_running_by_label "k8s-app=kube-dns" "kube-system"

sleep 5
# Cleanup

info "Deleting cluster $clustername..."
$EXE cluster delete "$clustername" || failed "could not delete the cluster $clustername"

highlight "[DONE] RegistryTest $EXTRA_TITLE"

exit 0


