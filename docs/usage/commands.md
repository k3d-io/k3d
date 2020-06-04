# Command Tree

```bash
k3d
  --runtime  # choose the container runtime (default: docker)
  --verbose  # enable verbose (debug) logging (default: false)
  create
    cluster [CLUSTERNAME]  # default cluster name is 'k3s-default'
      -a, --api-port  # specify the port on which the cluster will be accessible (e.g. via kubectl)
      -i, --image  # specify which k3s image should be used for the nodes
      --k3s-agent-arg  # add additional arguments to the k3s agent (see https://rancher.com/docs/k3s/latest/en/installation/install-options/agent-config/#k3s-agent-cli-help)
      --k3s-server-arg  # add additional arguments to the k3s server (see https://rancher.com/docs/k3s/latest/en/installation/install-options/server-config/#k3s-server-cli-help)
      -m, --masters  # specify how many master nodes you want to create
      --network  # specify a network you want to connect to
      --no-image-volume  # disable the creation of a volume for storing images (used for the 'k3d load image' command)
      -p, --port  # add some more port mappings
      --token  # specify a cluster token (default: auto-generated)
      --timeout  # specify a timeout, after which the cluster creation will be interrupted and changes rolled back
      --update-kubeconfig  # enable the automated update of the default kubeconfig with the details of the newly created cluster (also sets '--wait=true')
      --switch  # (implies --update-kubeconfig) automatically sets the current-context of your default kubeconfig to the new cluster's context
      -v, --volume  # specify additional bind-mounts
      --wait  # enable waiting for all master nodes to be ready before returning
      -w, --workers  # specify how many worker nodes you want to create
    node NODENAME  # Create new nodes (and add them to existing clusters)
      -c, --cluster  # specify the cluster that the node shall connect to
      -i, --image  # specify which k3s image should be used for the node(s)
          --replicas  # specify how many replicas you want to create with this spec
          --role  # specify the node role
      --wait  # wait for the node to be up and running before returning
      --timeout # specify a timeout duration, after which the node creation will be interrupted, if not done yet
  delete
    cluster CLUSTERNAME  # delete an existing cluster
      -a, --all  # delete all existing clusters
    node NODENAME  # delete an existing node
      -a, --all  # delete all existing nodes
  start
    cluster CLUSTERNAME  # start a (stopped) cluster
      -a, --all  # start all clusters
      --wait  # wait for all masters and master-loadbalancer to be up before returning
      --timeout  # maximum waiting time for '--wait' before canceling/returning
    node NODENAME  # start a (stopped) node
  stop
    cluster CLUSTERNAME  # stop a cluster
      -a, --all  # stop all clusters
    node  # stop a node
  get
    cluster [CLUSTERNAME [CLUSTERNAME ...]]
      --no-headers  # do not print headers
      --token  # show column with cluster tokens
    node NODENAME
      --no-headers  # do not print headers
    kubeconfig (CLUSTERNAME [CLUSTERNAME ...] | --all)
      -a, --all  # get kubeconfigs from all clusters
          --output  # specify the output file where the kubeconfig should be written to
          --overwrite  # [Careful!] forcefully overwrite the output file, ignoring existing contents
      -s, --switch  # switch current-context in kubeconfig to the new context
      -u, --update  # update conflicting fields in existing kubeconfig (default: true)
  load
    image  [IMAGE | ARCHIVE [IMAGE | ARCHIVE ...]]  # Load one or more images from the local runtime environment or tar-archives into k3d clusters
      -c, --cluster  # clusters to load the image into
      -k, --keep-tarball  # do not delete the image tarball from the shared volume after completion
  completion SHELL  # Generate completion scripts
  version  # show k3d build version
  help [COMMAND]  # show help text for any command
```
