## k3d cluster create

Create a new cluster

### Synopsis


Create a new k3s cluster with containerized nodes (k3s in docker).
Every cluster will consist of one or more containers:
	- 1 (or more) server node container (k3s)
	- (optionally) 1 loadbalancer container as the entrypoint to the cluster (nginx)
	- (optionally) 1 (or more) agent node containers (k3s)


```
k3d cluster create NAME [flags]
```

### Options

```
  -a, --agents int                                                     Specify how many agents you want to create
      --agents-memory string                                           Memory limit imposed on the agents nodes [From docker]
      --api-port [HOST:]HOSTPORT                                       Specify the Kubernetes API server port exposed on the LoadBalancer (Format: [HOST:]HOSTPORT)
                                                                        - Example: `k3d cluster create --servers 3 --api-port 0.0.0.0:6550`
  -c, --config string                                                  Path of a config file to use
  -e, --env KEY[=VALUE][@NODEFILTER[;NODEFILTER...]]                   Add environment variables to nodes (Format: KEY[=VALUE][@NODEFILTER[;NODEFILTER...]]
                                                                        - Example: `k3d cluster create --agents 2 -e "HTTP_PROXY=my.proxy.com" -e "SOME_KEY=SOME_VAL@server[0]"`
      --gpus string                                                    GPU devices to add to the cluster node containers ('all' to pass all GPUs) [From docker]
  -h, --help                                                           help for create
  -i, --image string                                                   Specify k3s image that you want to use for the nodes
      --k3s-agent-arg k3s agent                                        Additional args passed to the k3s agent command on agent nodes (new flag per arg)
      --k3s-server-arg k3s server                                      Additional args passed to the k3s server command on server nodes (new flag per arg)
      --kubeconfig-switch-context                                      Directly switch the default kubeconfig's current-context to the new cluster's context (requires --kubeconfig-update-default) (default true)
      --kubeconfig-update-default                                      Directly update the default kubeconfig with the new cluster's context (default true)
  -l, --label KEY[=VALUE][@NODEFILTER[;NODEFILTER...]]                 Add label to node container (Format: KEY[=VALUE][@NODEFILTER[;NODEFILTER...]]
                                                                        - Example: `k3d cluster create --agents 2 -l "my.label@agent[0,1]" -v "other.label=somevalue@server[0]"`
      --network string                                                 Join an existing network
      --no-hostip                                                      Disable the automatic injection of the Host IP as 'host.k3d.internal' into the containers and CoreDNS
      --no-image-volume                                                Disable the creation of a volume for importing images
      --no-lb                                                          Disable the creation of a LoadBalancer in front of the server nodes
      --no-rollback                                                    Disable the automatic rollback actions, if anything goes wrong
  -p, --port [HOST:][HOSTPORT:]CONTAINERPORT[/PROTOCOL][@NODEFILTER]   Map ports from the node containers to the host (Format: [HOST:][HOSTPORT:]CONTAINERPORT[/PROTOCOL][@NODEFILTER])
                                                                        - Example: `k3d cluster create --agents 2 -p 8080:80@agent[0] -p 8081@agent[1]`
      --registry-config string                                         Specify path to an extra registries.yaml file
      --registry-create                                                Create a k3d-managed registry and connect it to the cluster
      --registry-use stringArray                                       Connect to one or more k3d-managed registries running locally
  -s, --servers int                                                    Specify how many servers you want to create
      --servers-memory string                                          Memory limit imposed on the server nodes [From docker]
      --subnet 172.28.0.0/16                                           [Experimental: IPAM] Define a subnet for the newly created container network (Example: 172.28.0.0/16)
      --timeout duration                                               Rollback changes if cluster couldn't be created in specified duration.
      --token string                                                   Specify a cluster token. By default, we generate one.
  -v, --volume [SOURCE:]DEST[@NODEFILTER[;NODEFILTER...]]              Mount volumes into the nodes (Format: [SOURCE:]DEST[@NODEFILTER[;NODEFILTER...]]
                                                                        - Example: `k3d cluster create --agents 2 -v /my/path@agent[0,1] -v /tmp/test:/tmp/other@server[0]`
      --wait                                                           Wait for the server(s) to be ready before returning. Use '--timeout DURATION' to not wait forever. (default true)
```

### Options inherited from parent commands

```
      --timestamps   Enable Log timestamps
      --trace        Enable super verbose output (trace logging)
      --verbose      Enable verbose output (debug logging)
```

### SEE ALSO

* [k3d cluster](k3d_cluster.md)	 - Manage cluster(s)

