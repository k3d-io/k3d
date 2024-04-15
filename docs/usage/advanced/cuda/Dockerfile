ARG K3S_TAG="v1.28.8-k3s1"
ARG CUDA_TAG="12.4.1-base-ubuntu22.04"

FROM rancher/k3s:$K3S_TAG as k3s
FROM nvcr.io/nvidia/cuda:$CUDA_TAG

# Install the NVIDIA container toolkit
RUN apt-get update && apt-get install -y curl \
    && curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg \
    && curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
      sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
      tee /etc/apt/sources.list.d/nvidia-container-toolkit.list \
    && apt-get update && apt-get install -y nvidia-container-toolkit \
    && nvidia-ctk runtime configure --runtime=containerd

COPY --from=k3s / / --exclude=/bin
COPY --from=k3s /bin /bin

# Deploy the nvidia driver plugin on startup
COPY device-plugin-daemonset.yaml /var/lib/rancher/k3s/server/manifests/nvidia-device-plugin-daemonset.yaml

VOLUME /var/lib/kubelet
VOLUME /var/lib/rancher/k3s
VOLUME /var/lib/cni
VOLUME /var/log

ENV PATH="$PATH:/bin/aux"

ENTRYPOINT ["/bin/k3s"]
CMD ["agent"]