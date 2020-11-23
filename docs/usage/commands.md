# Command Tree

```bash
k3d
  --verbose  # enable verbose (debug) logging (default: false)
  --version  # show k3d and k3s version
  -h, --help  # show help text
  version  # show k3d and k3s version
  help [COMMAND]  # show help text for any command
  completion [bash | zsh | (psh | powershell)]  # generate completion scripts for common shells
  cluster [CLUSTERNAME]  # default cluster name is 'k3s-default'
    create
      --api-port  # specify the port on which the cluster will be accessible (e.g. via kubectl)
      -i, --image  # specify which k3s image should be used for the nodes
      --k3s-agent-arg  # add additional arguments to the k3s agent (see https://rancher.com/docs/k3s/latest/en/installation/install-options/agent-config/#k3s-agent-cli-help)
      --k3s-server-arg  # add additional arguments to the k3s server (see https://rancher.com/docs/k3s/latest/en/installation/install-options/server-config/#k3s-server-cli-help)
      -s, --servers  # specify how many server nodes you want to create
      --network  # specify a network you want to connect to
      --no-hostip # disable the automatic injection of the Host IP as 'host.k3d.internal' into the containers and CoreDN
      --no-image-volume  # disable the creation of a volume for storing images (used for the 'k3d load image' command)
      --no-lb # disable the creation of a LoadBalancer in front of the server nodes
      --no-rollback # disable the automatic rollback actions, if anything goes wrong
      -p, --port  # add some more port mappings
      --token  # specify a cluster token (default: auto-generated)
      --timeout  # specify a timeout, after which the cluster creation will be interrupted and changes rolled back
      --update-default-kubeconfig  # enable the automated update of the default kubeconfig with the details of the newly created cluster (also sets '--wait=true')
      --switch-context  # (implies --update-default-kubeconfig) automatically sets the current-context of your default kubeconfig to the new cluster's context
      -v, --volume  # specify additional bind-mounts
      --wait  # enable waiting for all server nodes to be ready before returning
      -a, --agents  # specify how many agent nodes you want to create
      -e, --env  # add environment variables to the node containers
    start CLUSTERNAME  # start a (stopped) cluster
      -a, --all  # start all clusters
      --wait  # wait for all servers and server-loadbalancer to be up before returning
      --timeout  # maximum waiting time for '--wait' before canceling/returning
    stop CLUSTERNAME  # stop a cluster
      -a, --all  # stop all clusters
    delete CLUSTERNAME  # delete an existing cluster
      -a, --all  # delete all existing clusters
    list [CLUSTERNAME [CLUSTERNAME ...]]
      --no-headers  # do not print headers
      --token  # show column with cluster tokens
  node
    create NODENAME  # Create new nodes (and add them to existing clusters)
      -c, --cluster  # specify the cluster that the node shall connect to
      -i, --image  # specify which k3s image should be used for the node(s)
          --replicas  # specify how many replicas you want to create with this spec
          --role  # specify the node role
      --wait  # wait for the node to be up and running before returning
      --timeout # specify a timeout duration, after which the node creation will be interrupted, if not done yet
    start NODENAME  # start a (stopped) node
    stop NODENAME # stop a node
    delete NODENAME  # delete an existing node
      -a, --all  # delete all existing nodes
    list NODENAME
      --no-headers  # do not print headers
  kubeconfig
    get (CLUSTERNAME [CLUSTERNAME ...] | --all) # get kubeconfig from cluster(s) and write it to stdout
      -a, --all  # get kubeconfigs from all clusters
    merge | write (CLUSTERNAME [CLUSTERNAME ...] | --all)  # get kubeconfig from cluster(s) and merge it/them into into a file in $HOME/.k3d (or whatever you specify via the flags)
      -a, --all  # get kubeconfigs from all clusters
          --output  # specify the output file where the kubeconfig should be written to
          --overwrite  # [Careful!] forcefully overwrite the output file, ignoring existing contents
      -s, --switch-context  # switch current-context in kubeconfig to the new context
      -u, --update  # update conflicting fields in existing kubeconfig (default: true)
      -d, --merge-default-kubeconfig  # update the default kubeconfig (usually $KUBECONFIG or $HOME/.kube/config)
  image
    import [IMAGE | ARCHIVE [IMAGE | ARCHIVE ...]]  # Load one or more images from the local runtime environment or tar-archives into k3d clusters
      -c, --cluster  # clusters to load the image into
      -k, --keep-tarball  # do not delete the image tarball from the shared volume after completion
```
