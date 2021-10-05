ARG K3S_TAG="v1.21.2-k3s1"
FROM rancher/k3s:$K3S_TAG as k3s

FROM nvidia/cuda:11.2.0-base-ubuntu18.04

ARG NVIDIA_CONTAINER_RUNTIME_VERSION
ENV NVIDIA_CONTAINER_RUNTIME_VERSION=$NVIDIA_CONTAINER_RUNTIME_VERSION

RUN echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections

RUN apt-get update && \
    apt-get -y install gnupg2 curl

# Install NVIDIA Container Runtime
RUN curl -s -L https://nvidia.github.io/nvidia-container-runtime/gpgkey | apt-key add -

RUN curl -s -L https://nvidia.github.io/nvidia-container-runtime/ubuntu18.04/nvidia-container-runtime.list | tee /etc/apt/sources.list.d/nvidia-container-runtime.list

RUN apt-get update && \
    apt-get -y install nvidia-container-runtime=${NVIDIA_CONTAINER_RUNTIME_VERSION}

COPY --from=k3s / /

RUN mkdir -p /etc && \
    echo 'hosts: files dns' > /etc/nsswitch.conf

RUN chmod 1777 /tmp

# Provide custom containerd configuration to configure the nvidia-container-runtime
RUN mkdir -p /var/lib/rancher/k3s/agent/etc/containerd/

COPY config.toml.tmpl /var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl

# Deploy the nvidia driver plugin on startup
RUN mkdir -p /var/lib/rancher/k3s/server/manifests

COPY device-plugin-daemonset.yaml /var/lib/rancher/k3s/server/manifests/nvidia-device-plugin-daemonset.yaml

VOLUME /var/lib/kubelet
VOLUME /var/lib/rancher/k3s
VOLUME /var/lib/cni
VOLUME /var/log

ENV PATH="$PATH:/bin/aux"

ENTRYPOINT ["/bin/k3s"]
CMD ["agent"]