# Running CUDA workloads

If you want to run CUDA workloads on the K3s container you need to customize the container.  
CUDA workloads require the NVIDIA Container Runtime, so containerd needs to be configured to use this runtime.  
The K3s container itself also needs to run with this runtime.  
If you are using Docker you can install the [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html).

## Building a customized K3s image

To get the NVIDIA container runtime in the K3s image you need to build your own K3s image.  
The native K3s image is based on Alpine but the NVIDIA container runtime is not supported on Alpine yet.  
To get around this we need to build the image with a supported base image.

### Dockerfiles
  
[Dockerfile.base](cuda/Dockerfile.base):

```Dockerfile
{% include "cuda/Dockerfile.base" %}

```  
  
[Dockerfile.k3d-gpu](cuda/Dockerfile.k3d-gpu):  

```Dockerfile
{% include "cuda/Dockerfile.k3d-gpu" %}
```

These Dockerfiles are based on the [K3s Dockerfile](https://github.com/rancher/k3s/blob/master/package/Dockerfile)
The following changes are applied:

1. Change the base images to nvidia/cuda:11.2.0-base-ubuntu18.04 so the NVIDIA Container Runtime can be installed. The version of `cuda:xx.x.x` must match the one you're planning to use.
2. Add a custom containerd `config.toml` template to add the NVIDIA Container Runtime. This replaces the default `runc` runtime
3. Add a manifest for the NVIDIA driver plugin for Kubernetes

### Configure containerd

We need to configure containerd to use the NVIDIA Container Runtime. We need to customize the config.toml that is used at startup. K3s provides a way to do this using a [config.toml.tmpl](cuda/config.toml.tmpl) file. More information can be found on the [K3s site](https://rancher.com/docs/k3s/latest/en/advanced/#configuring-containerd).

```go
{% include "cuda/config.toml.tmpl" %}
```

### The NVIDIA device plugin

To enable NVIDIA GPU support on Kubernetes you also need to install the [NVIDIA device plugin](https://github.com/NVIDIA/k8s-device-plugin). The device plugin is a deamonset and allows you to automatically:

* Expose the number of GPUs on each nodes of your cluster
* Keep track of the health of your GPUs
* Run GPU enabled containers in your Kubernetes cluster.

```yaml
{% include "cuda/gpu.yaml" %}
```

### Build the K3s image

To build the custom image we need to build K3s because we need the generated output.

Put the following files in a directory:

* [Dockerfile.base](cuda/Dockerfile.base)
* [Dockerfile.k3d-gpu](cuda/Dockerfile.k3d-gpu)
* [config.toml.tmpl](cuda/config.toml.tmpl)
* [gpu.yaml](cuda/gpu.yaml)
* [build.sh](cuda/build.sh)
* [cuda-vector-add.yaml](cuda/cuda-vector-add.yaml)

The `build.sh` script is configured using exports & defaults to `v1.21.2+k3s1`. Please set your CI_REGISTRY_IMAGE! The script performs the following steps:

* pulls K3s
* builds K3s
* build the custom K3D Docker image

The resulting image is tagged as k3s-gpu:&lt;version tag&gt;. The version tag is the git tag but the '+' sign is replaced with a '-'.

[build.sh](cuda/build.sh):

```bash
{% include "cuda/build.sh" %}
```

## Run and test the custom image with Docker

You can run a container based on the new image with Docker:

```bash
docker run --name k3s-gpu -d --privileged --gpus all $CI_REGISTRY_IMAGE:$IMAGE_TAG
```

Deploy a [test pod](cuda/cuda-vector-add.yaml):

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

Deploy a [test pod](cuda/cuda-vector-add.yaml):

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
* [K3s](https://github.com/rancher/k3s)
* [k3s-gpu](https://gitlab.com/vainkop1/k3s-gpu)

## Authors

* [@markrexwinkel](https://github.com/markrexwinkel)
* [@vainkop](https://github.com/vainkop)
* [@iwilltry42](https://github.com/iwilltry42)
