# Using Config Files

The config file feature is **available as of k3d v4.0.0**

## Introduction

!!! info "Syntax & Semantics"
    The options defined in the config file are not 100% the same as the CLI flags.  
    This concerns naming and style/usage/structure, e.g.

    - `--api-port` is split up into a field named `kubeAPI` that has 3 different "child fields" (`host`, `hostIP` and `hostPort`)
    - k3d options are bundled in a scope named `options.k3d`, where `--no-rollback` is defined as `options.k3d.disableRollback`
    - repeatable flags (like `--port`) are reflected as YAML lists

## Usage

Using a config file is as easy as putting it in a well-known place in your file system and then referencing it via flag:

- All options in config file: `k3d cluster create --config /home/me/my-awesome-config.yaml` (must be `.yaml`/`.yml`)
- With CLI override (name): `k3d cluster create somename --config /home/me/my-awesome-config.yaml`
- With CLI override (extra volume): `k3d cluster create --config /home/me/my-awesome-config.yaml --volume '/some/path:/some:path@server:0'`

## Required Fields

As of the time of writing this documentation, the config file only **requires** you to define two fields:

- `apiVersion` to match the version of the config file that you want to use (at this time it would be `apiVersion: k3d.io/v1alpha5`)
- `kind` to define the kind of config file that you want to use (currently we only have the `Simple` config)

So this would be the minimal config file, which configures absolutely nothing:

```yaml
apiVersion: k3d.io/v1alpha5
kind: Simple
```

## Config Options

The configuration options for k3d are continuously evolving and so is the config file (syntax) itself.  
Currently, the config file is still in an Alpha-State, meaning, that it is subject to change anytime (though we try to keep breaking changes low).

!!! info "Validation via JSON-Schema"
    k3d uses a [JSON-Schema](https://json-schema.org/) to describe the expected format and fields of the configuration file.  
    This schema is also used to [validate](https://github.com/xeipuuv/gojsonschema#validation) a user-given config file.  
    This JSON-Schema can be found in the specific config version sub-directory in the repository (e.g. [here for `v1alpha5`](https://github.com/k3d-io/k3d/blob/main/pkg/config/v1alpha5/schema.json)) and could be used to lookup supported fields or by linters to validate the config file, e.g. in your code editor.  

### All Options: Example

Since the config options and the config file are changing quite a bit, it's hard to keep track of all the supported config file settings, so here's an example showing all of them as of the time of writing:

```yaml
# k3d configuration file, saved as e.g. /home/me/myk3dcluster.yaml
apiVersion: k3d.io/v1alpha5 # this will change in the future as we make everything more stable
kind: Simple # internally, we also have a Cluster config, which is not yet available externally
metadata:
  name: mycluster # name that you want to give to your cluster (will still be prefixed with `k3d-`)
servers: 1 # same as `--servers 1`
agents: 2 # same as `--agents 2`
kubeAPI: # same as `--api-port myhost.my.domain:6445` (where the name would resolve to 127.0.0.1)
  host: "myhost.my.domain" # important for the `server` setting in the kubeconfig
  hostIP: "127.0.0.1" # where the Kubernetes API will be listening on
  hostPort: "6445" # where the Kubernetes API listening port will be mapped to on your host system
image: rancher/k3s:v1.20.4-k3s1 # same as `--image rancher/k3s:v1.20.4-k3s1`
network: my-custom-net # same as `--network my-custom-net`
subnet: "172.28.0.0/16" # same as `--subnet 172.28.0.0/16`
token: superSecretToken # same as `--token superSecretToken`
volumes: # repeatable flags are represented as YAML lists
  - volume: /my/host/path:/path/in/node # same as `--volume '/my/host/path:/path/in/node@server:0;agent:*'`
    nodeFilters:
      - server:0
      - agent:*
ports:
  - port: 8080:80 # same as `--port '8080:80@loadbalancer'`
    nodeFilters:
      - loadbalancer
env:
  - envVar: bar=baz # same as `--env 'bar=baz@server:0'`
    nodeFilters:
      - server:0
files:
  - description: 'Source: Embedded, Destination: Magic shortcut path'
    source: |
      apiVersion: v1
      kind: Namespace
      metadata:
        name: foo
    destination: k3s-manifests-custom/foo.yaml # Resolved to /var/lib/rancher/k3s/server/manifests/custom/foo.yaml
  - description: 'Source: Relative, Destination: Absolute path, Node: Servers only'
    source: ns-baz.yaml
    destination: /var/lib/rancher/k3s/server/manifests/baz.yaml
    nodeFilters:
    - "server:*"
registries: # define how registries should be created or used
  create: # creates a default registry to be used with the cluster; same as `--registry-create registry.localhost`
    name: registry.localhost
    host: "0.0.0.0"
    hostPort: "5000"
    proxy: # omit this to have a "normal" registry, set this to create a registry proxy (pull-through cache)
      remoteURL: https://registry-1.docker.io # mirror the DockerHub registry
      username: "" # unauthenticated
      password: "" # unauthenticated
    volumes:
      - /some/path:/var/lib/registry # persist registry data locally
  use:
    - k3d-myotherregistry:5000 # some other k3d-managed registry; same as `--registry-use 'k3d-myotherregistry:5000'`
  config: | # define contents of the `registries.yaml` file (or reference a file); same as `--registry-config /path/to/config.yaml`
    mirrors:
      "my.company.registry":
        endpoint:
          - http://my.company.registry:5000
hostAliases: # /etc/hosts style entries to be injected into /etc/hosts in the node containers and in the NodeHosts section in CoreDNS
  - ip: 1.2.3.4
    hostnames: 
      - my.host.local
      - that.other.local
  - ip: 1.1.1.1
    hostnames:
      - cloud.flare.dns
options:
  k3d: # k3d runtime settings
    wait: true # wait for cluster to be usable before returning; same as `--wait` (default: true)
    timeout: "60s" # wait timeout before aborting; same as `--timeout 60s`
    disableLoadbalancer: false # same as `--no-lb`
    disableImageVolume: false # same as `--no-image-volume`
    disableRollback: false # same as `--no-Rollback`
    loadbalancer:
      configOverrides:
        - settings.workerConnections=2048
  k3s: # options passed on to K3s itself
    extraArgs: # additional arguments passed to the `k3s server|agent` command; same as `--k3s-arg`
      - arg: "--tls-san=my.host.domain"
        nodeFilters:
          - server:*
    nodeLabels:
      - label: foo=bar # same as `--k3s-node-label 'foo=bar@agent:1'` -> this results in a Kubernetes node label
        nodeFilters:
          - agent:1
  kubeconfig:
    updateDefaultKubeconfig: true # add new cluster to your default Kubeconfig; same as `--kubeconfig-update-default` (default: true)
    switchCurrentContext: true # also set current-context to the new cluster's context; same as `--kubeconfig-switch-context` (default: true)
  runtime: # runtime (docker) specific options
    gpuRequest: all # same as `--gpus all`
    labels:
      - label: bar=baz # same as `--runtime-label 'bar=baz@agent:1'` -> this results in a runtime (docker) container label
        nodeFilters:
          - agent:1
    ulimits:
      - name: nofile
        soft: 26677
        hard: 26677

```

## Tips

- k3d [expands environment variables](https://pkg.go.dev/os#ExpandEnv) (`$VAR` or `${VAR}`) unconditionally in the config file, even before processing it in any way.  

## Config File vs. CLI Flags

k3d uses [`Cobra`](https://github.com/spf13/cobra) and [`Viper`](https://github.com/spf13/viper) for CLI and general config handling respectively.  
This automatically introduces a "config option order of priority" ([precedence order](https://github.com/spf13/viper#why-viper)):

!!! info "Config Precedence Order"
    Source: [spf13/viper#why-viper](https://github.com/spf13/viper#why-viper)  
    >Internal Setting > **CLI Flag** > Environment Variable > **Config File** > (k/v store >) Defaults

This means, that you can define e.g. a "base configuration file" with settings that you share across different clusters and override only the fields that differ between those clusters in your CLI flags/arguments.  
For example, you use the same config file to create three clusters which only have different names and `kubeAPI` (`--api-port`) settings.

## References

- k3d demo repository: <https://github.com/iwilltry42/k3d-demo/blob/main/README.md#config-file-support>
- SUSE Blog: <https://www.suse.com/c/introduction-k3d-run-k3s-docker-src/> (Search for `The “Configuration as Code” Way`)
