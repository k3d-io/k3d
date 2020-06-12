#!/bin/bash

K3D_EXE=${EXE:-/bin/k3d}
K3D_IMAGE_TAG=$1

# define E2E_KEEP to non-empty for keeping the e2e runner container after running the tests
E2E_KEEP=${E2E_KEEP:-}

# Max. time to wait for the runner container to be up
RUNNER_START_TIMEOUT=${E2E_RUNNER_START_TIMEOUT:-10}

####################################################################################

# Start the runner container
TIMESTAMP=$(date "+%y%m%d%H%M%S")
k3de2e=$(docker run -d \
          -v "$(pwd)/tests:/tests" \
          --privileged \
          -e EXE="$K3D_EXE" \
          -e CI="true" \
          -e LOG_LEVEL="$LOG_LEVEL" \
          -e E2E_SKIP="$E2E_SKIP" \
          --name "k3d-e2e-runner-$TIMESTAMP" \
          "k3d:$K3D_IMAGE_TAG")

# setup exit trap (make sure that we always stop and remove the runner container)
finish() {
  docker stop "$k3de2e" || /bin/true
  if [ -z "$E2E_KEEP" ] ; then
    docker rm "$k3de2e" || /bin/true
  fi
}
trap finish EXIT

# wait for the runner container to be up or exit early
TIMEOUT=0
until docker inspect "$k3de2e" | jq ".[0].State.Running" && docker logs "$k3de2e" 2>&1 | grep -i "API listen on /var/run/docker.sock"; do
  if [[ $TIMEOUT -eq $RUNNER_START_TIMEOUT ]]; then
    echo "Failed to start E2E Runner Container in $RUNNER_START_TIMEOUT seconds"
    exit 1
  fi
  sleep 1
  (( TIMEOUT++ ))
done

# execute tests
docker exec "$k3de2e" /tests/runner.sh
