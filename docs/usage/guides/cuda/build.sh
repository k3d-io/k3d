#!/bin/bash
set -e
cd $(dirname $0)
 
K3S_TAG="${1:-v1.18.10+k3s1}"
IMAGE_TAG="${K3S_TAG/+/-}"

if [ -d k3s ]; then
    rm -rf k3s
fi
git clone --depth 1 https://github.com/rancher/k3s.git -b $K3S_TAG
cd k3s
make
cd ..
docker build -t k3s-gpu:$IMAGE_TAG .