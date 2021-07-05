#!/bin/bash

export CI_REGISTRY_IMAGE="YOUR_REGISTRY_IMAGE_URL"
export VERSION="1.0"
export K3S_TAG="v1.21.2+k3s1"
export DOCKER_VERSION="20.10.7"
export IMAGE_TAG="v1.21.2-k3s1"
export NVIDIA_CONTAINER_RUNTIME_VERSION="3.5.0-1"

docker build -f Dockerfile.base --build-arg DOCKER_VERSION=$DOCKER_VERSION -t $CI_REGISTRY_IMAGE/base:$VERSION . && \
docker push $CI_REGISTRY_IMAGE/base:$VERSION

rm -rf ./k3s && \
git clone --depth 1 https://github.com/rancher/k3s.git -b "$K3S_TAG" && \
docker run -ti -v ${PWD}/k3s:/k3s -v /var/run/docker.sock:/var/run/docker.sock $CI_REGISTRY_IMAGE/base:1.0 sh -c "cd /k3s && make" && \
ls -al k3s/build/out/data.tar.zst

if [ -f k3s/build/out/data.tar.zst ]; then
  echo "File exists! Building!"
  docker build -f Dockerfile.k3d-gpu \
    --build-arg NVIDIA_CONTAINER_RUNTIME_VERSION=$NVIDIA_CONTAINER_RUNTIME_VERSION\
    -t $CI_REGISTRY_IMAGE:$IMAGE_TAG . && \
  docker push $CI_REGISTRY_IMAGE:$IMAGE_TAG
  echo "Done!"
else
  echo "Error, file does not exist!"
  exit 1
fi

docker build -t $CI_REGISTRY_IMAGE:$IMAGE_TAG .