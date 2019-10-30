# Examples

## Expose services

### 1. via Ingress

In this example, we will deploy a simple nginx webserver deployment and make it accessible via Iingress.
Therefore, we have to create the cluster in a way, that the internal port 80 (where the `traefik` ingress controller is listening on) is exposed on the host system.

1. Create a cluster, mapping the ingress port 80 to localhost:8081

    `k3d create --api-port 6550 --publish 8081:80 --workers 2`

    - Note: `--api-port 6550` is not required for the example to work. It's used to have `k3s`'s API-Server listening on port 6550 with that port mapped to the host system.

2. Get the kubeconfig file

    `export KUBECONFIG="$(k3d get-kubeconfig --name='k3s-default')"`

3. Create a nginx deployment

    `kubectl create deployment nginx --image=nginx`

4. Create a ClusterIP service for it

    `kubectl create service clusterip nginx --tcp=80:80`

5. Create an ingress object for it with `kubectl apply -f`
  *Note*: `k3s` deploys [`traefik`](https://github.com/containous/traefik) as the default ingress controller

    ```YAML
    apiVersion: extensions/v1beta1
    kind: Ingress
    metadata:
      name: nginx
      annotations:
        ingress.kubernetes.io/ssl-redirect: "false"
    spec:
      rules:
      - http:
          paths:
          - path: /
            backend:
              serviceName: nginx
              servicePort: 80
    ```

6. Curl it via localhost

    `curl localhost:8081/`

### 2. via NodePort

1. Create a cluster, mapping the port 30080 from worker-0 to localhost:8082

    `k3d create --publish 8082:30080@k3d-k3s-default-worker-0 --workers 2`

    - Note: Kubernetes' default NodePort range is [`30000-32767`](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport)

... (Steps 2 and 3 like above) ...

1. Create a NodePort service for it with `kubectl apply -f`

    ```YAML
    apiVersion: v1
    kind: Service
    metadata:
      labels:
        app: nginx
      name: nginx
    spec:
      ports:
      - name: 80-80
        nodePort: 30080
        port: 80
        protocol: TCP
        targetPort: 80
      selector:
        app: nginx
      type: NodePort
    ```

2. Curl it via localhost

    `curl localhost:8082/`

## Connect with a local insecure registry

This guide takes you through setting up a local insecure (http) registry and integrating it into your workflow so that:

- you can push to the registry from your host
- the cluster managed by k3d can pull from that registry

The registry will be named `registry.local` and run on port `5000`.

### Create the registry

<pre>
docker volume create local_registry

docker container run -d --name <b>registry.local</b> -v local_registry:/var/lib/registry --restart always -p <b>5000:5000</b> registry:2
</pre>

### Create the cluster with k3d

First we need a place to store the config template: `mkdir -p /home/${USER}/.k3d`

Create a file named `config.toml.tmpl` in `/home/${USER}/.k3d`, with following content:

<pre>
# Original section: no changes
[plugins.opt]
path = "{{ .NodeConfig.Containerd.Opt }}"
[plugins.cri]
stream_server_address = "{{ .NodeConfig.AgentConfig.NodeName }}"
stream_server_port = "10010"
{{- if .IsRunningInUserNS }}
disable_cgroup = true
disable_apparmor = true
restrict_oom_score_adj = true
{{ end -}}
{{- if .NodeConfig.AgentConfig.PauseImage }}
sandbox_image = "{{ .NodeConfig.AgentConfig.PauseImage }}"
{{ end -}}
{{- if not .NodeConfig.NoFlannel }}
  [plugins.cri.cni]
    bin_dir = "{{ .NodeConfig.AgentConfig.CNIBinDir }}"
    conf_dir = "{{ .NodeConfig.AgentConfig.CNIConfDir }}"
{{ end -}}

# Added section: additional registries and the endpoints
[plugins.cri.registry.mirrors]
  [plugins.cri.registry.mirrors."<b>registry.local:5000</b>"]
    endpoint = ["http://<b>registry.local:5000</b>"]
</pre>

Finally start a cluster with k3d, passing-in the config template:

```bash
CLUSTER_NAME=k3s-default
k3d create \
    --name ${CLUSTER_NAME} \
    --wait 0 \
    --auto-restart \
    --volume /home/${USER}/.k3d/config.toml.tmpl:/var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl
```

### Wire them up

- Connect the registry to the cluster network: `docker network connect k3d-k3s-default registry.local`
- Add `127.0.0.1 registry.local` to your `/etc/hosts`

### Test

Push an image to the registry:

```bash
docker pull nginx:latest
docker tag nginx:latest registry.local:5000/nginx:latest
docker push registry.local:5000/nginx:latest
```

Deploy a pod referencing this image to your cluster:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-test-registry
  labels:
    app: nginx-test-registry
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx-test-registry
  template:
    metadata:
      labels:
        app: nginx-test-registry
    spec:
      containers:
      - name: nginx-test-registry
        image: registry.local:5000/nginx:latest
        ports:
        - containerPort: 80
EOF
```

... and check that the pod is running: `kubectl get pods -l "app=nginx-test-registry"`

## Running on filesystems k3s doesn't like (btrfs, tmpfs, …)

The following script leverages a [Docker loopback volume plugin](https://github.com/ashald/docker-volume-loopback) to mask the problematic filesystem away from k3s by providing a small ext4 filesystem underneath `/var/lib/rancher/k3s` (k3s' data dir).

```bash
#!/bin/bash -x

CLUSTER_NAME="${1:-k3s-default}"
NUM_WORKERS="${2:-2}"

setup() {
  PLUGIN_LS_OUT=`docker plugin ls --format '{{.Name}},{{.Enabled}}' | grep -E '^ashald/docker-volume-loopback'`
  [ -z "${PLUGIN_LS_OUT}" ] && docker plugin install ashald/docker-volume-loopback DATA_DIR=/tmp/docker-loop/data
  sleep 3
  [ "${PLUGIN_LS_OUT##*,}" != "true" ] && docker plugin enable ashald/docker-volume-loopback

  K3D_MOUNTS=()
  for i in `seq 0 ${NUM_WORKERS}`; do
    [ ${i} -eq 0 ] && VOLUME_NAME="k3d-${CLUSTER_NAME}-server" || VOLUME_NAME="k3d-${CLUSTER_NAME}-worker-$((${i}-1))"
    docker volume create -d ashald/docker-volume-loopback ${VOLUME_NAME} -o sparse=true -o fs=ext4
    K3D_MOUNTS+=('-v' "${VOLUME_NAME}:/var/lib/rancher/k3s@${VOLUME_NAME}")
  done
  k3d c -i rancher/k3s:v0.9.1 -n ${CLUSTER_NAME} -w ${NUM_WORKERS} ${K3D_MOUNTS[@]}
}

cleanup() {
  K3D_VOLUMES=()
  k3d d -n ${CLUSTER_NAME}
  for i in `seq 0 ${NUM_WORKERS}`; do
    [ ${i} -eq 0 ] && VOLUME_NAME="k3d-${CLUSTER_NAME}-server" || VOLUME_NAME="k3d-${CLUSTER_NAME}-worker-$((${i}-1))"
    K3D_VOLUMES+=("${VOLUME_NAME}")
  done
  docker volume rm -f ${K3D_VOLUMES[@]}
}

setup
#cleanup
```
## Rancher for administering the cluster

Install rancher in to the same cluser for administering the cluster.

```
apiVersion: v1
kind: Namespace
metadata:
  name: cattle-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: rancher
  namespace: cattle-system

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  # "namespace" omitted since ClusterRoles are not namespaced
  name: rancher
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: rancher
subjects:
- kind: ServiceAccount
  name: rancher
  namespace: cattle-system
  apiGroup: ""
roleRef:
  kind: ClusterRole #this must be Role or ClusterRole
  name: rancher
  apiGroup: ""
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rancher
  namespace: cattle-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: rancher
  template:
    metadata:
      labels:
        app: rancher
    spec:
      serviceAccountName: rancher
      containers:
      - name: name
        image: rancher/rancher
        resources:
          requests:
            cpu: 100m
            memory: 200Mi
        ports:
        - containerPort: 443
---
apiVersion: v1
kind: Service
metadata:
  name: rancher
  namespace: cattle-system
spec:
  type: LoadBalancer
  ports:
  - port: 8443
    targetPort: 443
  selector:
    app: rancher
```
and then
`kubectl apply -f rancher.yaml`
