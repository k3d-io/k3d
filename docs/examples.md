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
