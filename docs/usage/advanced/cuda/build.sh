#!/usr/bin/env bash
# set -euxo pipefail

# Set default values
K3S_TAG=${K3S_TAG:="v1.31.7-k3s1"} # replace + with -, if needed
CUDA_TAG=${CUDA_TAG:="12.8.1-base-ubuntu24.04"}
#IMAGE_REGISTRY=${IMAGE_REGISTRY:="techmakers.azurecr.io"}
IMAGE_REGISTRY=${IMAGE_REGISTRY:="docker.io"}
IMAGE_REPOSITORY=${IMAGE_REPOSITORY:="k3s"}
IMAGE_TAG="${K3S_TAG//+/-}-cuda-$CUDA_TAG"
IMAGE=${IMAGE:="$IMAGE_REGISTRY/$IMAGE_REPOSITORY:$IMAGE_TAG"}

echo "IMAGE=$IMAGE"

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
  echo "Docker is not installed. Please install Docker first." >&2
  exit 1
fi

# Check if Docker service is running
if ! systemctl is-active --quiet docker; then
  echo "Docker service is not running. Attempting to start it..." >&2
  sudo systemctl start docker
fi

# Check if user is in docker group
if ! groups | grep -q '\bdocker\b'; then
  echo "WARNING: You are not in the 'docker' group. You may need to use sudo for docker commands."
fi

# Check if az CLI is installed
if ! command -v az &> /dev/null; then
  echo "Azure CLI (az) is not installed. Please install it first." >&2
  exit 1
fi

# Login to Azure container registry
# echo "Logging into Azure..."
# az acr login --name "$(echo $IMAGE_REGISTRY | cut -d. -f1)"

# --- Build and Push ---
echo "Building image..."
docker build --debug \
  --build-arg K3S_TAG=$K3S_TAG \
  --build-arg CUDA_TAG=$CUDA_TAG \
  -t "$IMAGE" .

# echo "Pushing image..."
# docker push "$IMAGE"

echo "Done!"
