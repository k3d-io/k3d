# Use Calico instead of Flannel

!!! info "Network Policies"
    k3s comes with a controller that enforces network policies by default.
    While You do not need to switch to any CNIs for Kubernetes network policies to be enforced, other CNIs such as Calico can help you to bridge the gap where Kubernetes network policies may lack some capabilities. See <https://github.com/k3s-io/k3s/issues/1308> for more information.  
    The docs below assume you want to switch to Calico's policy engine, thus setting `--disable-network-policy@server:*`.

## 1. Create the cluster without flannel
By default K3s deploys flannel CNI to take care of networking in your environment.
Since we want to use Calico in this example we have to disable the default CNI.
This can be done by using the `--k3s-arg` flag at the cluster creation time.  

Use the following command to create your cluster:
```bash
k3d cluster create "${clustername}" \
  --k3s-arg '--flannel-backend=none@server:*' \
  --k3s-arg '--disable-network-policy@server:*' \
  --k3s-arg '--cluster-cidr=192.168.0.0/16@server:*'
```

In this example :

- Change the `"${clustername}"` with the name of the cluster (or set a variable).
- Cluster will use the "192.168.0.0/16" CIDR, if you want to change the default CIDR make sure to change it in the `custom-resources.yaml` too.

## 2. Install Calico 
A simple way to install Calico is to use the Tigera Operator.
The operator helps us to configure, install and upgrade Calico in an environment.

Use the following command to install the operator:
```bash
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.29.0/manifests/tigera-operator.yaml
```

The operator periodically checks for the installation manifest.
This manifest is how we instruct the Tigera Operator to install Calico.

Use the following command to create the installation manifest:
```bash
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.29.0/manifests/custom-resources.yaml
```

At this point, our installation is in progress, and we can verify it by using the following command:
```bash
kubectl get tigerastatus
```

After a minute, you should see a result similar to the following:
```
NAME        AVAILABLE   PROGRESSING   DEGRADED   SINCE
apiserver   True        False         False      30s
calico      True        False         False      10s
ippools     True        False         False      70s
```

Great Calico is up and running!

## 3. IP forwarding
By default, Calico disables IP forwarding inside the containers.
This can cause an issue in some cases where you are using load balancers. You can learn more about loadblanacers [here](https://docs.k3s.io/networking/networking-services#service-load-balancer).
To fix this issue we have to turn on the IP forwarding flag inside `calico-node` pods.

Use the following command to enable forwarding via the operator:
```bash
kubectl patch installation default --type=merge --patch='{"spec":{"calicoNetwork":{"containerIPForwarding":"Enabled"}}}'
```

## 4. What's next?
Check out our other guides, here some suggestions:
- Add an additional node to your setup. [see](../commands/k3d_node.md)
- Expose your services. [see](../exposing_services.md)

## References

- <https://rancher.com/docs/k3s/latest/en/installation/network-options/>  
- <https://docs.tigera.io/calico/latest/getting-started/kubernetes/k3s/quickstart>
