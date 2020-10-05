# Use Calico instead of Flannel
If you want to use NetworkPolicy you can use Calico in k3s instead of Flannel.

## 1. Disable Flannel
On the k3s cluster creation add the flag `--flannel-backend=none`. For this on k3d you need to foward this flag to k3s with the option `--k3s-server-arg`.

So the command of the cluster creation is
```bash
    k3d cluster create "${clustername}" --k3s-server-arg --flannel-backend=none
```
In this exemple change `"${clustername}"` with the name of the cluster (or set a variable). You can add other options, [see](../commands.md).  

The cluster will start without flannel (and without other CNI).

## 2. Install Calico
Now you need to install Calico 

### 2.1. Download and modify the Calico descriptor
You can following the [documentation](https://docs.projectcalico.org/master/reference/cni-plugin/configuration)

And then you have to change the ConfigMap `calico-config`. On the `cni_network_config` add the entry for allowing IP forwarding  
```json
    "container_settings": {
        "allow_ip_forwarding": true
    }
```
Or directly using the descriptor calico.yaml


### 2.2. Apply the descriptor
With kubectl (check the usage `kubectl kubeconfig` if you can't connect to the cluster).

From the root of the repository
```bash
    kubectl apply -f docs/usage/guides/calico.yaml
```

And watch for the calico pod(s) deployment
```
    watch "kubectl get pods -n kube-system"
```
<!> Calico is not as quick as Flannel (but it provides more features)

## References
https://rancher.com/docs/k3s/latest/en/installation/network-options/  
https://docs.projectcalico.org/getting-started/kubernetes/k3s/
