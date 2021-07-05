# Running CUDA workloads

If you want to run CUDA workloads on the K3S container you need to customize the container.  
CUDA workloads require the NVIDIA Container Runtime, so containerd needs to be configured to use this runtime.  
The K3S container itself also needs to run with this runtime.  
If you are using Docker you can install the [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html).

## Building a customized K3S image

To get the NVIDIA container runtime in the K3S image you need to build your own K3S image.  
The native K3S image is based on Alpine but the NVIDIA container runtime is not supported on Alpine yet.  
To get around this we need to build the image with a supported base image.

### Dockerfiles:  
  
Dockerfile.base:
```Dockerfile
FROM nvidia/cuda:11.2.0-base-ubuntu18.04

ENV DEBIAN_FRONTEND noninteractive

ARG DOCKER_VERSION
ENV DOCKER_VERSION=$DOCKER_VERSION

RUN set -x && \
    apt-get update && \
    apt-get install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    wget \
    tar \
    zstd \
    gnupg \
    lsb-release \
    git \
    software-properties-common \
    build-essential && \
    rm -rf /var/lib/apt/lists/*

RUN set -x && \
    curl -fsSL https://download.docker.com/linux/$(lsb_release -is | tr '[:upper:]' '[:lower:]')/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg && \
    echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/$(lsb_release -is | tr '[:upper:]' '[:lower:]') $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null && \
    apt-get update && \
    apt-get install -y \
    containerd.io \
    docker-ce=5:$DOCKER_VERSION~3-0~$(lsb_release -is | tr '[:upper:]' '[:lower:]')-$(lsb_release -cs) \
    docker-ce-cli=5:$DOCKER_VERSION~3-0~$(lsb_release -is | tr '[:upper:]' '[:lower:]')-$(lsb_release -cs) && \
    rm -rf /var/lib/apt/lists/*

```  
  
  
  
Dockerfile.k3d-gpu:  

```Dockerfile
FROM nvidia/cuda:11.2.0-base-ubuntu18.04 as base

RUN set -x && \
    apt-get update && \
    apt-get install -y ca-certificates zstd

COPY k3s/build/out/data.tar.zst /

RUN set -x && \
    mkdir -p /image/etc/ssl/certs /image/run /image/var/run /image/tmp /image/lib/modules /image/lib/firmware && \
    tar -I zstd -xf /data.tar.zst -C /image && \
    cp /etc/ssl/certs/ca-certificates.crt /image/etc/ssl/certs/ca-certificates.crt

RUN set -x && \
    cd image/bin && \
    rm -f k3s && \
    ln -s k3s-server k3s

FROM nvidia/cuda:11.2.0-base-ubuntu18.04

RUN set -x && \
    echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections

RUN set -x && \
    apt-get update && \
    apt-get -y install gnupg2 curl

# Install NVIDIA Container Runtime
RUN set -x && \
    curl -s -L https://nvidia.github.io/nvidia-container-runtime/gpgkey | apt-key add -

RUN set -x && \
    curl -s -L https://nvidia.github.io/nvidia-container-runtime/ubuntu18.04/nvidia-container-runtime.list | tee /etc/apt/sources.list.d/nvidia-container-runtime.list

RUN set -x && \
    apt-get update && \
    apt-get -y install nvidia-container-runtime=3.5.0-1


COPY --from=base /image /

RUN set -x && \
    mkdir -p /etc && \
    echo 'hosts: files dns' > /etc/nsswitch.conf

RUN set -x && \
    chmod 1777 /tmp

# Provide custom containerd configuration to configure the nvidia-container-runtime
RUN set -x && \
    mkdir -p /var/lib/rancher/k3s/agent/etc/containerd/

COPY config.toml.tmpl /var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl

# Deploy the nvidia driver plugin on startup
RUN set -x && \
    mkdir -p /var/lib/rancher/k3s/server/manifests

COPY gpu.yaml /var/lib/rancher/k3s/server/manifests/gpu.yaml

VOLUME /var/lib/kubelet
VOLUME /var/lib/rancher/k3s
VOLUME /var/lib/cni
VOLUME /var/log

ENV PATH="$PATH:/bin/aux"

ENTRYPOINT ["/bin/k3s"]
CMD ["agent"]
```

These Dockerfiles [Dockerfile.base](https://github.com/vainkop/k3d/blob/main/docs/usage/guides/cuda/Dockerfile.base) + [Dockerfile.k3d-gpu](https://github.com/vainkop/k3d/blob/main/docs/usage/guides/cuda/Dockerfile.k3d-gpu) are based on the [K3s Dockerfile](https://github.com/rancher/k3s/blob/master/package/Dockerfile)
The following changes are applied:

1. Change the base images to nvidia/cuda:11.2.0-base-ubuntu18.04 so the NVIDIA Container Runtime can be installed. The version of `cuda:xx.x.x` must match the one you're planning to use.
2. Add a custom containerd `config.toml` template to add the NVIDIA Container Runtime. This replaces the default `runc` runtime
3. Add a manifest for the NVIDIA driver plugin for Kubernetes

### Configure containerd

We need to configure containerd to use the NVIDIA Container Runtime. We need to customize the config.toml that is used at startup. K3s provides a way to do this using a [config.toml.tmpl](cuda/config.toml.tmpl) file. More information can be found on the [K3s site](https://rancher.com/docs/k3s/latest/en/advanced/#configuring-containerd).

```go
[plugins.opt]
  path = "{{ .NodeConfig.Containerd.Opt }}"

[plugins.cri]
  stream_server_address = "127.0.0.1"
  stream_server_port = "10010"

{{- if .IsRunningInUserNS }}
  disable_cgroup = true
  disable_apparmor = true
  restrict_oom_score_adj = true
{{end}}

{{- if .NodeConfig.AgentConfig.PauseImage }}
  sandbox_image = "{{ .NodeConfig.AgentConfig.PauseImage }}"
{{end}}

{{- if not .NodeConfig.NoFlannel }}
[plugins.cri.cni]
  bin_dir = "{{ .NodeConfig.AgentConfig.CNIBinDir }}"
  conf_dir = "{{ .NodeConfig.AgentConfig.CNIConfDir }}"
{{end}}

[plugins.cri.containerd.runtimes.runc]
  # ---- changed from 'io.containerd.runc.v2' for GPU support
  runtime_type = "io.containerd.runtime.v1.linux"

# ---- added for GPU support
[plugins.linux]
  runtime = "nvidia-container-runtime"

{{ if .PrivateRegistryConfig }}
{{ if .PrivateRegistryConfig.Mirrors }}
[plugins.cri.registry.mirrors]{{end}}
{{range $k, $v := .PrivateRegistryConfig.Mirrors }}
[plugins.cri.registry.mirrors."{{$k}}"]
  endpoint = [{{range $i, $j := $v.Endpoints}}{{if $i}}, {{end}}{{printf "%q" .}}{{end}}]
{{end}}

{{range $k, $v := .PrivateRegistryConfig.Configs }}
{{ if $v.Auth }}
[plugins.cri.registry.configs."{{$k}}".auth]
  {{ if $v.Auth.Username }}username = "{{ $v.Auth.Username }}"{{end}}
  {{ if $v.Auth.Password }}password = "{{ $v.Auth.Password }}"{{end}}
  {{ if $v.Auth.Auth }}auth = "{{ $v.Auth.Auth }}"{{end}}
  {{ if $v.Auth.IdentityToken }}identitytoken = "{{ $v.Auth.IdentityToken }}"{{end}}
{{end}}
{{ if $v.TLS }}
[plugins.cri.registry.configs."{{$k}}".tls]
  {{ if $v.TLS.CAFile }}ca_file = "{{ $v.TLS.CAFile }}"{{end}}
  {{ if $v.TLS.CertFile }}cert_file = "{{ $v.TLS.CertFile }}"{{end}}
  {{ if $v.TLS.KeyFile }}key_file = "{{ $v.TLS.KeyFile }}"{{end}}
{{end}}
{{end}}
{{end}}
```

### The NVIDIA device plugin

To enable NVIDIA GPU support on Kubernetes you also need to install the [NVIDIA device plugin](https://github.com/NVIDIA/k8s-device-plugin). The device plugin is a deamonset and allows you to automatically:

* Expose the number of GPUs on each nodes of your cluster
* Keep track of the health of your GPUs
* Run GPU enabled containers in your Kubernetes cluster.

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: nvidia-device-plugin-daemonset
  namespace: kube-system
spec:
  selector:
    matchLabels:
      name: nvidia-device-plugin-ds
  template:
    metadata:
      # Mark this pod as a critical add-on; when enabled, the critical add-on scheduler
      # reserves resources for critical add-on pods so that they can be rescheduled after
      # a failure.  This annotation works in tandem with the toleration below.
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ""
      labels:
        name: nvidia-device-plugin-ds
    spec:
      tolerations:
      # Allow this pod to be rescheduled while the node is in "critical add-ons only" mode.
      # This, along with the annotation above marks this pod as a critical add-on.
      - key: CriticalAddonsOnly
        operator: Exists
      containers:
      - env:
        - name: DP_DISABLE_HEALTHCHECKS
          value: xids
        image: nvidia/k8s-device-plugin:1.11
        name: nvidia-device-plugin-ctr
        securityContext:
          allowPrivilegeEscalation: true
          capabilities:
            drop: ["ALL"]
        volumeMounts:
          - name: device-plugin
            mountPath: /var/lib/kubelet/device-plugins
      volumes:
        - name: device-plugin
          hostPath:
            path: /var/lib/kubelet/device-plugins
```

### Build the K3S image

To build the custom image we need to build K3S because we need the generated output.

Put the following files in a directory:
* [Dockerfile.base](https://github.com/vainkop/k3d/blob/main/docs/usage/guides/cuda/Dockerfile.base)
* [Dockerfile.k3d-gpu](https://github.com/vainkop/k3d/blob/main/docs/usage/guides/cuda/Dockerfile.k3d-gpu)
* [config.toml.tmpl](cuda/config.toml.tmpl)
* [gpu.yaml](https://github.com/vainkop/k3d/blob/main/docs/usage/guides/cuda/gpu.yaml)
* [build.sh](https://github.com/vainkop/k3d/blob/main/docs/usage/guides/cuda/build.sh)
* [cuda-vector-add.yaml](https://github.com/vainkop/k3d/blob/main/docs/usage/guides/cuda/cuda-vector-add.yaml)

The `build.sh` script is configured using exports & defaults to `v1.21.2+k3s1`. Please set your CI_REGISTRY_IMAGE! The script performs the following steps:

* pulls K3S
* builds K3S
* build the custom K3D Docker image

The resulting image is tagged as k3s-gpu:&lt;version tag&gt;. The version tag is the git tag but the '+' sign is replaced with a '-'.

[build.sh](https://github.com/vainkop/k3d/blob/main/docs/usage/guides/cuda/build.sh):

```bash
#!/bin/bash

export CI_REGISTRY_IMAGE="YOUR_REGISTRY_IMAGE_URL"
export VERSION="1.0"
export K3S_TAG="v1.21.2+k3s1"
export DOCKER_VERSION="20.10.7"
export IMAGE_TAG="v1.21.2-k3s1"

sudo docker build -f Dockerfile.base --build-arg DOCKER_VERSION=$DOCKER_VERSION -t $CI_REGISTRY_IMAGE/base:$VERSION . && \
sudo docker push $CI_REGISTRY_IMAGE/base:$VERSION

sudo rm -rf ./k3s && \
git clone --depth 1 https://github.com/rancher/k3s.git -b "$K3S_TAG" && \
sudo docker run -ti -v ${PWD}/k3s:/k3s -v /var/run/docker.sock:/var/run/docker.sock $CI_REGISTRY_IMAGE/base:1.0 sh -c "cd /k3s && make" && \
sudo ls -al k3s/build/out/data.tar.zst

if [ -f k3s/build/out/data.tar.zst ]; then
  echo "File exists! Building!"
  sudo docker build -f Dockerfile.k3d-gpu -t $CI_REGISTRY_IMAGE:$IMAGE_TAG . && \
  sudo docker push $CI_REGISTRY_IMAGE:$IMAGE_TAG
  echo "Done!"
else
  echo "Error, file does not exist!"
  exit 1
fi

docker build -t $CI_REGISTRY_IMAGE:$IMAGE_TAG .
```

## Run and test the custom image with Docker

You can run a container based on the new image with Docker:

```bash
docker run --name k3s-gpu -d --privileged --gpus all $CI_REGISTRY_IMAGE:$IMAGE_TAG
```

Deploy a [test pod](https://github.com/vainkop/k3d/blob/main/docs/usage/guides/cuda/cuda-vector-add.yaml):

```bash
docker cp cuda-vector-add.yaml k3s-gpu:/cuda-vector-add.yaml
docker exec k3s-gpu kubectl apply -f /cuda-vector-add.yaml
docker exec k3s-gpu kubectl logs cuda-vector-add
```

## Run and test the custom image with k3d

Tou can use the image with k3d:

```bash
k3d cluster create local --image=$CI_REGISTRY_IMAGE:$IMAGE_TAG --gpus=1
```

Deploy a [test pod](https://github.com/vainkop/k3d/blob/main/docs/usage/guides/cuda/cuda-vector-add.yaml):

```bash
kubectl apply -f cuda-vector-add.yaml
kubectl logs cuda-vector-add
```

## Known issues

* This approach does not work on WSL2 yet. The NVIDIA driver plugin and container runtime rely on the NVIDIA Management Library (NVML) which is not yet supported. See the [CUDA on WSL User Guide](https://docs.nvidia.com/cuda/wsl-user-guide/index.html#known-limitations).

## Acknowledgements

Most of the information in this article was obtained from various sources:

* [Add NVIDIA GPU support to k3s with containerd](https://dev.to/mweibel/add-nvidia-gpu-support-to-k3s-with-containerd-4j17)
* [microk8s](https://github.com/ubuntu/microk8s)
* [K3S](https://github.com/rancher/k3s)
* [k3s-gpu](https://gitlab.com/vainkop1/k3s-gpu)

## Authors

- [@markrexwinkel](https://github.com/markrexwinkel)
- [@vainkop](https://github.com/vainkop)
