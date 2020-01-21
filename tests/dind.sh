#!/bin/sh

K3D_IMAGE_TAG=$1

k3de2e=$(docker run -d --rm \
          -v "$(pwd)"/tests:/tests \
          --privileged \
          -e EXE="/bin/k3d" \
          -e CI="true" \
          k3d:$K3D_IMAGE_TAG)

sleep 5 # wait 5 seconds for docker to start

# Execute tests
docker exec $k3de2e /tests/runner.sh