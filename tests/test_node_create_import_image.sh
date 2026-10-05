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

export CURRENT_STAGE="Test | NodeCreateImportImage"

highlight "[START] NodeCreateImportImageTest"

clustername="nodeimporttest"

# a tag that only exists locally, so any pod using it with imagePullPolicy: Never
# proves the image came from --import-image rather than a registry pull
testimage="k3d-node-import-e2e:local"
testtar="$TEST_OUTPUT_DIR/k3d-node-import-e2e.tar"

info "Preparing local-only test image $testimage..."
docker pull busybox:1.36 > /dev/null || failed "could not pull busybox:1.36"
docker tag busybox:1.36 "$testimage" > /dev/null || failed "could not tag test image"
docker save -o "$testtar" "$testimage" || failed "could not save test image to $testtar"

info "Creating cluster $clustername..."
$EXE cluster create "$clustername" --timeout 360s --wait || failed "could not create cluster $clustername"

check_clusters "$clustername" || failed "error checking cluster"

# a pod spec that fails fast if the image isn't already present on its node
make_pod_yaml() {
  cat <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: $1
spec:
  nodeSelector:
    kubernetes.io/hostname: $2
  containers:
  - name: proof
    image: $testimage
    imagePullPolicy: Never
    command: ["sh", "-c", "sleep 3600"]
EOF
}

### 1. control: node created without --import-image must NOT have the image ###
info "Creating control node without --import-image..."
$EXE node create ctrl --cluster "$clustername" --role agent --wait --timeout 120s || failed "could not create control node"
check_multi_node "$clustername" 2 || failed "control node did not join the cluster"

info "Scheduling a Never-pull pod on the control node..."
make_pod_yaml "proof-ctrl" "k3d-ctrl-0" | kubectl apply -f - || failed "could not create control pod"

info "Waiting for the control pod to report ErrImageNeverPull..."
ctrlOk=false
for _ in $(seq 1 30); do
  reason=$(kubectl get pod proof-ctrl -o jsonpath='{.status.containerStatuses[0].state.waiting.reason}' 2>/dev/null)
  if [[ "$reason" == "ErrImageNeverPull" ]]; then
    ctrlOk=true
    break
  fi
  sleep 2
done
$ctrlOk || failed "control pod did not end up in ErrImageNeverPull (image was unexpectedly present without --import-image)"
passed "control node correctly lacks the image without --import-image"

### 2. --import-image <image-name>, resolved from the runtime ###
info "Creating node with --import-image $testimage..."
$EXE node create imp-name --cluster "$clustername" --role agent --import-image "$testimage" --wait --timeout 120s || failed "could not create node with --import-image (image name)"
check_multi_node "$clustername" 3 || failed "imp-name node did not join the cluster"

make_pod_yaml "proof-imp-name" "k3d-imp-name-0" | kubectl apply -f - || failed "could not create proof-imp-name pod"
wait_for_pod_running_by_name "proof-imp-name" || failed "image imported via image name was not available on the node"
passed "node created with --import-image <image-name> has the image available"

### 3. --import-image <tarball path> ###
info "Creating node with --import-image $testtar..."
$EXE node create imp-tar --cluster "$clustername" --role agent --import-image "$testtar" --wait --timeout 120s || failed "could not create node with --import-image (tarball)"
check_multi_node "$clustername" 4 || failed "imp-tar node did not join the cluster"

make_pod_yaml "proof-imp-tar" "k3d-imp-tar-0" | kubectl apply -f - || failed "could not create proof-imp-tar pod"
wait_for_pod_running_by_name "proof-imp-tar" || failed "image imported via tarball path was not available on the node"
passed "node created with --import-image <tarball-path> has the image available"

### 4. --replicas with --import-image: every replica gets the image from a single export ###
info "Creating 2 replicas with --import-image..."
$EXE node create imp-rep --cluster "$clustername" --role agent --replicas 2 --import-image "$testimage" --wait --timeout 120s || failed "could not create replica nodes with --import-image"
check_multi_node "$clustername" 6 || failed "replica nodes did not join the cluster"

for i in 0 1; do
  make_pod_yaml "proof-imp-rep-$i" "k3d-imp-rep-$i" | kubectl apply -f - || failed "could not create proof-imp-rep-$i pod"
  wait_for_pod_running_by_name "proof-imp-rep-$i" || failed "replica $i did not have the imported image"
done
passed "all replicas created with --import-image have the image available"

### 5. --import-image with an image that doesn't exist anywhere: hard failure, no node created ###
info "Creating node with a nonexistent --import-image (expected to fail)..."
if $EXE node create imp-bad --cluster "$clustername" --role agent --import-image "doesnotexist:v1" --wait --timeout 60s; then
  failed "node create with a nonexistent --import-image unexpectedly succeeded"
fi
check_multi_node "$clustername" 6 || failed "a node was created despite the invalid --import-image"
passed "node create with a nonexistent --import-image fails and creates no node"

# Cleanup

info "Deleting cluster $clustername..."
$EXE cluster delete "$clustername" || failed "could not delete the cluster $clustername"

rm -f "$testtar"
docker rmi "$testimage" > /dev/null 2>&1

exit 0
