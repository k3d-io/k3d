# K3s Features in k3d

K3s ships with lots of built-in features and services, some of which may only be used in "non-normal" ways in k3d due to the fact that K3s is running in containers.

## General: K3s documentation

- Automatically Deploying Manifests and Helm Charts: <https://rancher.com/docs/k3s/latest/en/helm/#automatically-deploying-manifests-and-helm-charts>
  - Note: `/var/lib/rancher/k3s/server/manifests` is also the path inside the K3s container filesystem, where all built-in component manifests are, so you can override them or provide your own variants by mounting files there, e.g. `--volume /path/to/my/custom/coredns.yaml:/var/lib/rancher/k3s/server/manifests/coredns.yaml` will override the packaged CoreDNS component.
- Customizing packaged Components with `HelmChartConfig`: <https://rancher.com/docs/k3s/latest/en/helm/#customizing-packaged-components-with-helmchartconfig>

## CoreDNS

> Cluster DNS service

### Resources

- Manifest embedded in K3s: <https://github.com/k3s-io/k3s/blob/master/manifests/coredns.yaml>
  - Note: it includes template variables (like `%{CLUSTER_DOMAIN}%`) that will be replaced by K3s before writing the file to the filesystem

### CoreDNS in k3d

Basically, CoreDNS works the same in k3d as it does in other clusters.
One thing to note though is, that the default `forward . /etc/resolv.conf` configured in the `Corefile` doesn't work the same, as the `/etc/resolv.conf` file inside the K3s node containers is not the same as the one on your local machine.

#### Modifications

As of k3d v5.x, k3d injects entries to the `NodeHosts` (basically a hosts file similar to `/etc/hosts` in Linux, which is managed by K3s) to enable Pods in the cluster to resolve the names of other containers in the same docker network (cluster network) and a special entry called `host.k3d.internal` which resolves to the IP of the network gateway (can be used to e.g. resolve DNS queries using your local resolver).
There's a PR in progress to make customizations easier (for k3d and for users): <https://github.com/k3s-io/k3s/pull/4397>

## local-path-provisioner

> Dynamically provisioning persistent local storage with Kubernetes

### Resources

- Source: <https://github.com/rancher/local-path-provisioner>
- Manifest embedded in K3s: <https://github.com/k3s-io/k3s/blob/master/manifests/local-storage.yaml>

### local-path-provisioner in k3d

In k3d, the local paths that the `local-path-provisioner` uses (default is `/var/lib/rancher/k3s/storage`) lies inside the container's filesystem, meaning that by default it's not mapped somewhere e.g. in your user home directory for you to use.
You'd need to map some local directory to that path to easily use the files inside this path: add `--volume $HOME/some/directory:/var/lib/rancher/k3s/storage@all` to your `k3d cluster create` command.

## Traefik

> Kubernetes Ingress Controller

### Resources

- Official Documentation: <https://doc.traefik.io/traefik/providers/kubernetes-ingress/>
- Manifest embedded in K3s: <https://github.com/k3s-io/k3s/blob/master/manifests/traefik.yaml>

### Traefik in k3d

k3d runs K3s in containers, so you'll need to expose the http/https ports on your host to easily access Ingress resources in your cluster. We have a guide over here explaining how to do this, [see](exposing_services.md)

## servicelb (klipper-lb)

> Embedded service load balancer in Klipper
> Allows you to use services with `type: LoadBalancer` in K3s by creating tiny proxies that use `hostPort`s

### Resources

- Source: <https://github.com/k3s-io/klipper-lb>

### servicelb in k3d

`klipper-lb` creates new pods that proxy traffic from `hostPort`s to the service ports of `type: LoadBalancer`.
The `hostPort` in this case is a port in a K3s container, not your local host, so you'd need to add the port-mapping via the `--port` flag when creating the cluster.
