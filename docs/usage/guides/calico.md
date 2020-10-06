# Use Calico instead of Flannel
If you want to use NetworkPolicy you can use Calico in k3s instead of Flannel.

### 1. Download and modify the Calico descriptor
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
- add the flag `--flannel-backend=none`. For this, on k3d you need to forward this flag to k3s with the option `--k3s-server-arg`.
- mount (`--volume`) the calico descriptor in the auto deploy manifest directory of k3s `/var/lib/rancher/k3s/server/manifests/`

So the command of the cluster creation is (when you are at root of the k3d repository)
```bash
    k3d cluster create "${clustername}" --k3s-server-arg '--flannel-backend=none' --volume "$(pwd)/docs/usage/guides/calico.yaml:/var/lib/rancher/k3s/server/manifests/calico.yaml"
```
In this example :
- change `"${clustername}"` with the name of the cluster (or set a variable). 
- `$(pwd)/docs/usage/guides/calico.yaml` is the absolute path of the calico manifest, you can adapt it.

You can add other options, [see](../commands.md).  

The cluster will start without flannel (and without other CNI).

For watching for the pod(s) deployment
```
    watch "kubectl get pods -n kube-system"
```

Note : 
- you can use the auto deploy manifest or a kubectl apply depending on your needs
- <!> Calico is not as quick as Flannel (but it provides more features)

## References
https://rancher.com/docs/k3s/latest/en/installation/network-options/  
https://docs.projectcalico.org/getting-started/kubernetes/k3s/
