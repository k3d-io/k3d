# Running CUDA workloads
If you want to run CUDA workloads on the K3S container you need to customize the container.
CUDA workloads require the NVIDIA Container Runtime, so containerd needs to be configured to use this runtime.
The K3S container itself also needs to run with this runtime. If you are using Docker you can install the [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html).

## Building a customized K3S image
To get the NVIDIA container runtime in the K3S image you need to build your own K3S image. The native K3S image is based on Alpine but the NVIDIA container runtime is not supported on Alpine yet. To get around this we need to build the image with a supported base image.

### Adapt the Dockerfile

```Dockerfile
FROM ubuntu:18.04 as base
RUN apt-get update -y && apt-get install -y ca-certificates
ADD k3s/build/out/data.tar.gz /image
RUN mkdir -p /image/etc/ssl/certs /image/run /image/var/run /image/tmp /image/lib/modules /image/lib/firmware && \
    cp /etc/ssl/certs/ca-certificates.crt /image/etc/ssl/certs/ca-certificates.crt
RUN cd image/bin && \
    rm -f k3s && \
    ln -s k3s-server k3s

FROM ubuntu:18.04
RUN echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections
RUN apt-get update -y && apt-get -y install gnupg2 curl

# Install the NVIDIA Container Runtime
RUN curl -s -L https://nvidia.github.io/nvidia-container-runtime/gpgkey | apt-key add -
RUN curl -s -L https://nvidia.github.io/nvidia-container-runtime/ubuntu18.04/nvidia-container-runtime.list | tee /etc/apt/sources.list.d/nvidia-container-runtime.list
RUN apt-get update -y
RUN apt-get -y install nvidia-container-runtime

COPY --from=base /image /
RUN mkdir -p /etc && \
    echo 'hosts: files dns' > /etc/nsswitch.conf
RUN chmod 1777 /tmp
# Provide custom containerd configuration to configure the nvidia-container-runtime
RUN mkdir -p /var/lib/rancher/k3s/agent/etc/containerd/
COPY config.toml.tmpl /var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl
# Deploy the nvidia driver plugin on startup
RUN mkdir -p /var/lib/rancher/k3s/server/manifests
COPY gpu.yaml /var/lib/rancher/k3s/server/manifests/gpu.yaml
VOLUME /var/lib/kubelet
VOLUME /var/lib/rancher/k3s
VOLUME /var/lib/cni
VOLUME /var/log
ENV PATH="$PATH:/bin/aux"
ENTRYPOINT ["/bin/k3s"]
CMD ["agent"]
```
This [Dockerfile](cuda/Dockerfile) is based on the [K3S Dockerfile](https://github.com/rancher/k3s/blob/master/package/Dockerfile).
The following changes are applied:
1. Change the base images to Ubuntu 18.04 so the NVIDIA Container Runtime can be installed
2. Add a custom containerd `config.toml` template to add the NVIDIA Container Runtime. This replaces the default `runc` runtime
3. Add a manifest for the NVIDIA driver plugin for Kubernetes

### Configure containerd
We need to configure containerd to use the NVIDIA Container Runtime. We need to customize the config.toml that is used at startup. K3S provides a way to do this using a [config.toml.tmpl](cuda/config.toml.tmpl) file. More information can be found on the [K3S site](https://rancher.com/docs/k3s/latest/en/advanced/#configuring-containerd).

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
* [Dockerfile](cuda/Dockerfile)
* [config.toml.tmpl](cuda/config.toml.tmpl)
* [gpu.yaml](cuda/gpu.yaml)
* [build.sh](cuda/build.sh)
* [cuda-vector-add.yaml](cuda/cuda-vector-add.yaml)

The `build.sh` files takes the K3S git tag as argument, it defaults to `v1.18.10+k3s1`. The script performs the following steps: 
* pulls K3S 
* builds K3S
* build the custom K3S Docker image

The resulting image is tagged as k3s-gpu:&lt;version tag&gt;. The version tag is the git tag but the '+' sign is replaced with a '-'.

[build.sh](cuda/build.sh):
```bash
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
```

## Run and test the custom image with Docker
You can run a container based on the new image with Docker:
```
docker run --name k3s-gpu -d --privileged --gpus all k3s-gpu:v1.18.10-k3s1
```
Deploy a [test pod](cuda/cuda-vector-add.yaml):
```
docker cp cuda-vector-add.yaml k3s-gpu:/cuda-vector-add.yaml
docker exec k3s-gpu kubectl apply -f /cuda-vector-add.yaml
docker exec k3s-gpu kubectl logs cuda-vector-add
```

## Run and test the custom image with k3d
Tou can use the image with k3d:
```
k3d cluster create --no-lb --image k3s-gpu:v1.18.10-k3s1 --gpus all
```
Deploy a [test pod](cuda/cuda-vector-add.yaml):
```
kubectl apply -f cuda-vector-add.yaml
kubectl logs cuda-vector-add
```

## Known issues
* This approach does not work on WSL2 yet. The NVIDIA driver plugin and container runtime rely on the NVIDIA Management Library (NVML) which is not yet supported. See the [CUDA on WSL User Guide](https://docs.nvidia.com/cuda/wsl-user-guide/index.html#known-limitations).

## Acknowledgements:
Most of the information in this article was obtained from various sources:
* [Add NVIDIA GPU support to k3s with containerd](https://dev.to/mweibel/add-nvidia-gpu-support-to-k3s-with-containerd-4j17)
* [microk8s](https://github.com/ubuntu/microk8s)
* [K3S](https://github.com/rancher/k3s)


