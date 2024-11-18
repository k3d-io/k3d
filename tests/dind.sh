#!/bin/bash

K3D_EXE=${EXE:-/bin/k3d}
K3D_IMAGE_TAG="$1-dind"

# define E2E_KEEP to non-empty for keeping the e2e runner container after running the tests
E2E_KEEP=${E2E_KEEP:-}

# Max. number of tests executed in parallel
E2E_PARALLEL=${E2E_PARALLEL:-}

# Max. time to wait for the runner container to be up
RUNNER_START_TIMEOUT=${E2E_RUNNER_START_TIMEOUT:-10}

# Override Docker-in-Docker version
E2E_DIND_VERSION=${E2E_DIND_VERSION:-}

# Fail on first error instead of waiting until all tests are done. Useful in CI.
E2E_FAIL_FAST=${E2E_FAIL_FAST:-}

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

info "Building container image k3d:${K3D_IMAGE_TAG}"
# Extra Image Build Arg
: "${EXTRA_BUILD_FLAG:=""}"
if [[ -n "$E2E_DIND_VERSION" ]]; then
  info "Testing with Docker version $E2E_DIND_VERSION"
  EXTRA_BUILD_FLAG="--build-arg DOCKER_VERSION=$E2E_DIND_VERSION"
fi
DOCKER_BUILDKIT=1 docker build . --quiet --no-cache $EXTRA_BUILD_FLAG --tag "k3d:${K3D_IMAGE_TAG}" --target "dind" || failed "Build failed"

# Start the runner container
TIMESTAMP=$(date "+%y%m%d%H%M%S")
container_name="k3d-e2e-runner-$TIMESTAMP"
info "Starting E2E Test Runner container ${container_name}"
k3de2e=$(docker run -d \
          -v "$(pwd):/src" \
          --privileged \
          -e EXE="$K3D_EXE" \
          -e CI="true" \
          -e LOG_LEVEL="$LOG_LEVEL" \
          -e E2E_FAIL_FAST="$E2E_FAIL_FAST" \
          -e E2E_INCLUDE="$E2E_INCLUDE" \
          -e E2E_EXCLUDE="$E2E_EXCLUDE" \
          -e E2E_PARALLEL="$E2E_PARALLEL" \
          -e E2E_EXTRA="$E2E_EXTRA" \
          -e K3S_IMAGE="$E2E_K3S_VERSION" \
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
  docker exec "$k3de2e" git config --global --add safe.directory /src
  docker exec --workdir /src "$k3de2e" make -j2 build-helper-images || failed "build-helper-images failed"
  # execute tests
  echo "Start time outside runner: $(date)"
  docker exec "$k3de2e" /src/tests/runner.sh
else
  # execute tests
  docker exec -e "K3D_HELPER_IMAGE_TAG=$E2E_HELPER_IMAGE_TAG" "$k3de2e" /src/tests/runner.sh
fi


