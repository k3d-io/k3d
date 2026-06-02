ARG K3S_TAG="v1.31.7-k3s1"
ARG CUDA_TAG="12.8.1-base-ubuntu24.04"
ARG NVIDIA_DRIVER_VERS="570"

# Stage 1: Pull k3s base image
FROM rancher/k3s:${K3S_TAG} AS k3s
# Nothing else needed here except the base

# Stage 2: CUDA + NVIDIA Toolkit layer
FROM nvcr.io/nvidia/cuda:${CUDA_TAG}

# Re-declare all ARGs you want to use in this stage
ARG NVIDIA_DRIVER_VERS

# Optional: useful for runtime debugging
ENV NVIDIA_DRIVER_VERS=${NVIDIA_DRIVER_VERS}
ENV PAGER=less

# Install NVIDIA container toolkit & matching utilities
# Install NVIDIA container toolkit, utilities, and graphics-drivers PPA
# YES WE KNOW we are at Ubuntu 24.04 but NVidia container toolkit does not support 24.04 YET
# https://github.com/NVIDIA/nvidia-container-toolkit/issues/482
RUN apt-get update && apt-get install -y \
    curl \
    gnupg \
    less \
    ca-certificates \
    software-properties-common \
    lsb-release \
    less \
    && curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | \
       gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg \
    && curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
       sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
       tee /etc/apt/sources.list.d/nvidia-container-toolkit.list \
    && apt-get update \
    && apt-get install -y \
       nvidia-container-toolkit \
       nvidia-utils-${NVIDIA_DRIVER_VERS}-server \
    && nvidia-ctk runtime configure --runtime=containerd \
    && apt-get autoremove -y && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Copy full K3s filesystem, split /bin for layering purposes
COPY --from=k3s / / --exclude=/bin
COPY --from=k3s /bin /bin

# Inject NVIDIA plugin manifest
COPY device-plugin-daemonset.yaml /var/lib/rancher/k3s/server/manifests/nvidia-device-plugin-daemonset.yaml

# Define required volumes for K3s runtime
VOLUME /var/lib/kubelet
VOLUME /var/lib/rancher/k3s
VOLUME /var/lib/cni
VOLUME /var/log

# Set path and defaults
ENV PATH="$PATH:/bin/aux"
ENTRYPOINT ["/bin/k3s"]
CMD ["agent"]
