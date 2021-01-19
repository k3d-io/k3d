# Command Tree

```bash
k3d
  --verbose  # GLOBAL: enable verbose (debug) logging (default: false)
  --trace  # GLOBAL: enable super verbose logging (trace logging) (default: false)
  --version  # show k3d and k3s version
  -h, --help  # GLOBAL: show help text

  cluster [CLUSTERNAME]  # default cluster name is 'k3s-default'
    create
      -a, --agents  # specify how many agent nodes you want to create (integer, default: 0)
      --api-port  # specify the port on which the cluster will be accessible (format '[HOST:]HOSTPORT', default: random)
      -c, --config  # use a config file (format 'PATH')
      -e, --env  # add environment variables to the nodes (quoted string, format: 'KEY[=VALUE][@NODEFILTER[;NODEFILTER...]]', use flag multiple times)
      --gpus  # [from docker CLI] add GPU devices to the node containers (string, e.g. 'all')
      -i, --image  # specify which k3s image should be used for the nodes (string, default: 'docker.io/rancher/k3s:v1.20.0-k3s2', tag changes per build)
      --k3s-agent-arg  # add additional arguments to the k3s agent (quoted string, use flag multiple times) (see https://rancher.com/docs/k3s/latest/en/installation/install-options/agent-config/#k3s-agent-cli-help)
      --k3s-server-arg  # add additional arguments to the k3s server (quoted string, use flag multiple times) (see https://rancher.com/docs/k3s/latest/en/installation/install-options/server-config/#k3s-server-cli-help)
      --kubeconfig-switch-context  # (implies --kubeconfig-update-default) automatically sets the current-context of your default kubeconfig to the new cluster's context (default: true)
      --kubeconfig-update-default  # enable the automated update of the default kubeconfig with the details of the newly created cluster (also sets '--wait=true') (default: true)
      -l, --label  # add (docker) labels to the node containers (format: 'KEY[=VALUE][@NODEFILTER[;NODEFILTER...]]', use flag multiple times)
      --network  # specify an existing (docker) network you want to connect to (string)
      --no-hostip  # disable the automatic injection of the Host IP as 'host.k3d.internal' into the containers and CoreDNS (default: false)
      --no-image-volume  # disable the creation of a volume for storing images (used for the 'k3d image import' command) (default: false)
      --no-lb  # disable the creation of a load balancer in front of the server nodes (default: false)
      --no-rollback  # disable the automatic rollback actions, if anything goes wrong (default: false)
      -p, --port  # add some more port mappings (format: '[HOST:][HOSTPORT:]CONTAINERPORT[/PROTOCOL][@NODEFILTER]', use flag multiple times)
      --registry-create  # create a new (docker) registry dedicated for this cluster (default: false)
      --registry-use  # use an existing local (docker) registry with this cluster (string, use multiple times)
      -s, --servers  # specify how many server nodes you want to create (integer, default: 1)
      --token  # specify a cluster token (string, default: auto-generated)
      --timeout  # specify a timeout, after which the cluster creation will be interrupted and changes rolled back (duration, e.g. '10s')
      -v, --volume  # specify additional bind-mounts (format: '[SOURCE:]DEST[@NODEFILTER[;NODEFILTER...]]', use flag multiple times)
      --wait  # enable waiting for all server nodes to be ready before returning (default: true)
    start CLUSTERNAME  # start a (stopped) cluster
      -a, --all  # start all clusters (default: false)
      --wait  # wait for all servers and server-loadbalancer to be up before returning (default: true)
      --timeout  # maximum waiting time for '--wait' before canceling/returning (duration, e.g. '10s')
    stop CLUSTERNAME  # stop a cluster
      -a, --all  # stop all clusters (default: false)
    delete CLUSTERNAME  # delete an existing cluster
      -a, --all  # delete all existing clusters (default: false)
    list [CLUSTERNAME [CLUSTERNAME ...]]
      --no-headers  # do not print headers (default: false)
      --token  # show column with cluster tokens (default: false)
      -o, --output  # format the output (format: 'json|yaml')
  completion [bash | zsh | fish | (psh | powershell)]  # generate completion scripts for common shells
  config
    init  # write a default k3d config (as a starting point)
      -f, --force  # force overwrite target file (default: false)
      -o, --output  # file to write to (string, default "k3d-default.yaml")
  help [COMMAND]  # show help text for any command
  image
    import [IMAGE | ARCHIVE [IMAGE | ARCHIVE ...]]  # Load one or more images from the local runtime environment or tar-archives into k3d clusters
      -c, --cluster  # clusters to load the image into (string, use flag multiple times, default: k3s-default)
      -k, --keep-tarball  # do not delete the image tarball from the shared volume after completion (default: false)
  kubeconfig
    get (CLUSTERNAME [CLUSTERNAME ...] | --all) # get kubeconfig from cluster(s) and write it to stdout
      -a, --all  # get kubeconfigs from all clusters (default: false)
    merge | write (CLUSTERNAME [CLUSTERNAME ...] | --all)  # get kubeconfig from cluster(s) and merge it/them into a (kubeconfig-)file
      -a, --all  # get kubeconfigs from all clusters (default: false)
      -s, --kubeconfig-switch-context  # switch current-context in kubeconfig to the new context (default: true)
      -d, --kubeconfig-merge-default  # update the default kubeconfig (usually $KUBECONFIG or $HOME/.kube/config)
      -o, --output  # specify the output file where the kubeconfig should be written to (string)
      --overwrite  # [Careful!] forcefully overwrite the output file, ignoring existing contents (default: false)
      -u, --update  # update conflicting fields in existing kubeconfig (default: true)
  node
    create NODENAME  # Create new nodes (and add them to existing clusters)
      -c, --cluster  # specify the cluster that the node shall connect to (string, default: k3s-default)
      -i, --image  # specify which k3s image should be used for the node(s) (string, default: 'docker.io/rancher/k3s:v1.20.0-k3s2', tag changes per build)
      --replicas  # specify how many replicas you want to create with this spec (integer, default: 1)
      --role  # specify the node role (string, format: 'agent|server', default: agent)
      --timeout # specify a timeout duration, after which the node creation will be interrupted, if not done yet (duration, e.g. '10s')
      --wait  # wait for the node to be up and running before returning (default: true)
    start NODENAME  # start a (stopped) node
    stop NODENAME # stop a node
    delete NODENAME  # delete an existing node
      -a, --all  # delete all existing nodes (default: false)
      -r, --registries  # also delete registries, as a special type of node (default: false)
    list NODENAME
      --no-headers  # do not print headers (default: false)
  registry
    create REGISTRYNAME
      -i, --image  # specify image used for the registry (string, default: "docker.io/library/registry:2")
      -p, --port  # select host port to map to (format: '[HOST:]HOSTPORT', default: 'random')
    delete REGISTRYNAME
      -a, --all  # delete all existing registries (default: false)
    list [NAME [NAME...]]
      --no-headers  # disable table headers (default: false)
  version  # show k3d and k3s version
```
