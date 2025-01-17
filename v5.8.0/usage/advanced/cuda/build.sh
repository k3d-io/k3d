#!/bin/bash

set -euxo pipefail

K3S_TAG=${K3S_TAG:="v1.28.8-k3s1"} # replace + with -, if needed
CUDA_TAG=${CUDA_TAG:="12.4.1-base-ubuntu22.04"}
IMAGE_REGISTRY=${IMAGE_REGISTRY:="MY_REGISTRY"}
IMAGE_REPOSITORY=${IMAGE_REPOSITORY:="rancher/k3s"}
IMAGE_TAG="$K3S_TAG-cuda-$CUDA_TAG"
IMAGE=${IMAGE:="$IMAGE_REGISTRY/$IMAGE_REPOSITORY:$IMAGE_TAG"}

echo "IMAGE=$IMAGE"

docker build \
  --build-arg K3S_TAG=$K3S_TAG \
  --build-arg CUDA_TAG=$CUDA_TAG \
  -t $IMAGE .
docker push $IMAGE
echo "Done!"