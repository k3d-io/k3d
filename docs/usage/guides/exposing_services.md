# Exposing Services

## 1. via Ingress

In this example, we will deploy a simple nginx webserver deployment and make it accessible via ingress.
Therefore, we have to create the cluster in a way, that the internal port 80 (where the `traefik` ingress controller is listening on) is exposed on the host system.

1. Create a cluster, mapping the ingress port 80 to localhost:8081

    `#!bash k3d cluster create --api-port 6550 -p 8081:80@loadbalancer --agents 2`

    !!! info "Good to know"
        - `--api-port 6550` is not required for the example to work. It's used to have `k3s`'s API-Server listening on port 6550 with that port mapped to the host system.
        - the port-mapping construct `8081:80@loadbalancer` means
            - map port `8081` from the host to port `80` on the container which matches the nodefilter `loadbalancer`
        - the `loadbalancer` nodefilter matches only the `serverlb` that's deployed in front of a cluster's server nodes
            - all ports exposed on the `serverlb` will be proxied to the same ports on all server nodes in the cluster

2. Get the kubeconfig file

    `#!bash export KUBECONFIG="$(k3d kubeconfig get k3s-default)"`

3. Create a nginx deployment

    `#!bash kubectl create deployment nginx --image=nginx`

4. Create a ClusterIP service for it

    `#!bash kubectl create service clusterip nginx --tcp=80:80`

5. Create an ingress object for it with `#!bash kubectl apply -f`
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

    `#!bash curl localhost:8081/`

## 2. via NodePort

1. Create a cluster, mapping the port 30080 from agent-0 to localhost:8082

    `#!bash k3d cluster create mycluster -p 8082:30080@agent[0] --agents 2`

    - Note: Kubernetes' default NodePort range is [`30000-32767`](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport)

... (Steps 2 and 3 like above) ...

1. Create a NodePort service for it with `#!bash kubectl apply -f`

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

    `#!bash curl localhost:8082/`
