# Use Calico instead of Flannel

Note: k3s comes with a controller that enforces network policies by default. You do not need to switch to Calico for network policies to be enforced. See <https://github.com/k3s-io/k3s/issues/1308> for more information.

## 1. Download and modify the Calico descriptor

You can following the [documentation](https://docs.projectcalico.org/master/reference/cni-plugin/configuration)

And then you have to change the ConfigMap `calico-config`. On the `cni_network_config` add the entry for allowing IP forwarding  

```json
"container_settings": {
    "allow_ip_forwarding": true
}
```

Or you can directly use this [calico.yaml](calico.yaml) manifest

## 2. Create the cluster without flannel and with calico

On the k3s cluster creation :

- add the flags `--flannel-backend=none` and `--disable-network-policy`. For this, on k3d you need to forward this flag to k3s with the option `--k3s-arg`.
- mount (`--volume`) the calico descriptor in the auto deploy manifest directory of k3s `/var/lib/rancher/k3s/server/manifests/`

So the command of the cluster creation is (when you are at root of the k3d repository)

```bash
k3d cluster create "${clustername}" \
  --k3s-arg '--flannel-backend=none@server:*' \
  --k3s-arg '--disable-network-policy' \
  --volume "$(pwd)/docs/usage/guides/calico.yaml:/var/lib/rancher/k3s/server/manifests/calico.yaml"
```

In this example :

- change `"${clustername}"` with the name of the cluster (or set a variable).
- `$(pwd)/docs/usage/guides/calico.yaml` is the absolute path of the calico manifest, you can adapt it.

You can add other options, [see](../commands.md).  

The cluster will start without flannel and with Calico as CNI Plugin.

For watching for the pod(s) deployment

```bash
watch "kubectl get pods -n kube-system"    
```

You will have something like this at beginning (with the command line `#!bash kubectl get pods -n kube-system`)

```bash
NAME                                       READY   STATUS     RESTARTS   AGE
helm-install-traefik-pn84f                 0/1     Pending    0          3s
calico-node-97rx8                          0/1     Init:0/3   0          3s
metrics-server-7566d596c8-hwnqq            0/1     Pending    0          2s
calico-kube-controllers-58b656d69f-2z7cn   0/1     Pending    0          2s
local-path-provisioner-6d59f47c7-rmswg     0/1     Pending    0          2s
coredns-8655855d6-cxtnr                    0/1     Pending    0          2s
```

And when it finish to start

```bash
NAME                                       READY   STATUS      RESTARTS   AGE
metrics-server-7566d596c8-hwnqq            1/1     Running     0          56s
calico-node-97rx8                          1/1     Running     0          57s
helm-install-traefik-pn84f                 0/1     Completed   1          57s
svclb-traefik-lmjr5                        2/2     Running     0          28s
calico-kube-controllers-58b656d69f-2z7cn   1/1     Running     0          56s
local-path-provisioner-6d59f47c7-rmswg     1/1     Running     0          56s
traefik-758cd5fc85-x8p57                   1/1     Running     0          28s
coredns-8655855d6-cxtnr                    1/1     Running     0          56s
```

Note :

- you can use the auto deploy manifest or a kubectl apply depending on your needs
- :exclamation: Calico is not as quick as Flannel (but it provides more features)

## References

- <https://rancher.com/docs/k3s/latest/en/installation/network-options/>  
- <https://docs.projectcalico.org/getting-started/kubernetes/k3s/>
