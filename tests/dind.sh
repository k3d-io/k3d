#!/bin/bash

K3D_EXE=${EXE:-/bin/k3d}
K3D_IMAGE_TAG=$1

# define E2E_KEEP to non-empty for keeping the e2e runner container after running the tests
E2E_KEEP=${E2E_KEEP:-}

# Max. time to wait for the runner container to be up
RUNNER_START_TIMEOUT=${E2E_RUNNER_START_TIMEOUT:-10}

####################################################################################

CURR_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
[ -d "$CURR_DIR" ] || {
  echo "FATAL: no current dir (maybe running in zsh?)"
  exit 1
}

export CURRENT_STAGE="DIND"


# shellcheck disable=SC1091
source "$CURR_DIR/common.sh"

####################################################################################

# Start the runner container
TIMESTAMP=$(date "+%y%m%d%H%M%S")
container_name="k3d-e2e-runner-$TIMESTAMP"
k3de2e=$(docker run -d \
          -v "$(pwd):/src" \
          --privileged \
          -e EXE="$K3D_EXE" \
          -e CI="true" \
          -e LOG_LEVEL="$LOG_LEVEL" \
          -e E2E_INCLUDE="$E2E_INCLUDE" \
          -e E2E_EXCLUDE="$E2E_EXCLUDE" \
          -e E2E_EXTRA="$E2E_EXTRA" \
          -e LOG_TIMESTAMPS="true" \
          --add-host "k3d-registrytest-registry:127.0.0.1" \
          --name "$container_name" \
          "k3d:$K3D_IMAGE_TAG")

# setup exit trap (make sure that we always stop and remove the runner container)
finish() {
  if [ -z "$E2E_KEEP" ] ; then
    info "Cleaning up test container $container_name"
    docker rm -f "$k3de2e" || /bin/true
  fi
}
trap finish EXIT

# wait for the runner container to be up or exit early
TIMEOUT=0
until docker inspect "$k3de2e" | jq ".[0].State.Running" && docker logs "$k3de2e" 2>&1 | grep -qi "API listen on /var/run/docker.sock"; do
  if [[ $TIMEOUT -eq $RUNNER_START_TIMEOUT ]]; then
    echo "Failed to start E2E Runner Container in $RUNNER_START_TIMEOUT seconds"
    exit 1
  fi
  sleep 1
  (( TIMEOUT++ ))
done

# build helper container images
if [ -z "$E2E_HELPER_IMAGE_TAG" ]; then
  docker exec --workdir /src "$k3de2e" make -j2 build-helper-images
  # execute tests
  echo "Start time outside runner: $(date)"
  docker exec "$k3de2e" /src/tests/runner.sh
else
  # execute tests
  docker exec -e "K3D_HELPER_IMAGE_TAG=$E2E_HELPER_IMAGE_TAG" "$k3de2e" /src/tests/runner.sh
fi


