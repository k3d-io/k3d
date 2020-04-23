# Using registries with k3d

## <a name="registries-file"></a>Registries configuration file

You can add registries by specifying them in a `registries.yaml` in your `$HOME/.k3d` directory.
This file will be loaded automatically by k3d if present and will be shared between all your
k3d clusters, but you can also use a specific file for a new cluster with the
`--registries-file` argument.

This file is a regular [k3s registries configuration file](https://rancher.com/docs/k3s/latest/en/installation/airgap/#create-registry-yaml),
and looks like this:

```yaml
mirrors:
  "my.company.registry:5000":
    endpoint:
      - http://my.company.registry:5000
```

In this example, an image with a name like `my.company.registry:5000/nginx:latest` would be
_pulled_ from the registry running at `http://my.company.registry:5000`. 

Note well there is an important limitation: **this configuration file will only work with
k3s >= v0.10.0**. It will fail silently with previous versions of k3s, but you find in the
[section below](#k3s-old) an alternative solution.

This file can also be used for providing additional information necessary for accessing
some registries, like [authentication](#auth) and [certificates](#certs).

### <a name="auth"></a>Authenticated registries

When using authenticated registries, we can add the _username_ and _password_ in a
`configs` section in the `registries.yaml`, like this:

```yaml
mirrors:
  my.company.registry:
    endpoint:
      - http://my.company.registry

configs:
  my.company.registry:
    auth:
      username: aladin
      password: abracadabra
```

### <a name="certs"></a>Secure registries

When using secure registries, the [`registries.yaml` file](#registries-file) must include information
about the certificates. For example, if you want to use images from the secure registry
running at `https://my.company.registry`, you must first download a CA file valid for that server
and store it in some well-known directory like `${HOME}/.k3d/my-company-root.pem`.  

Then you have to mount the CA file in some directory in the nodes in the cluster and
include that mounted file in a `configs` section in the [`registries.yaml` file](#registries-file).
For example, if we mount the CA file in `/etc/ssl/certs/my-company-root.pem`, the `registries.yaml`
will look like:

```yaml
mirrors:
  my.company.registry:
    endpoint:
      - https://my.company.registry

configs:
  my.company.registry:
    tls:
      # we will mount "my-company-root.pem" in the /etc/ssl/certs/ directory.
      ca_file: "/etc/ssl/certs/my-company-root.pem"
```

Finally, we can create the cluster, mounting the CA file in the path we
specified in `ca_file`:

```shell script
k3d create --volume ${HOME}/.k3d/my-company-root.pem:/etc/ssl/certs/my-company-root.pem ...
```

## Using a local registry

### Using the k3d registry

k3d can manage a local registry that you can use for pushing your images to, and your k3d nodes
will be able to use those images automatically. k3d will create the registry for you and connect
it to your k3d cluster. It is important to note that this registry will be shared between all
your k3d clusters, and it will be released when the last of the k3d clusters that was using it 
is deleted.

In order to enable the k3d registry when creating a new cluster, you must run k3d with the 
`--enable-registry` argument

```shell script
k3d create --enable-registry ...
```

Then you must make it accessible as described in [the next section](#etc-hosts). And
then you should [check your local registry](#testing).

### Using your own local registry

If you don't want k3d to manage your registry, you can start it with some `docker` commands, like:

```shell script
docker volume create local_registry
docker container run -d --name registry.localhost -v local_registry:/var/lib/registry --restart always -p 5000:5000 registry:2
```

These commands will start your registry in `registry.localhost:5000`. In order to push to this registry, you will
need to make it accessible as we described in [the next section ](#etc-hosts). Once your
registry is up and running, we will need to add it to your [`registries.yaml` configuration file](#registries-file).
Finally, you must connect the registry network to the k3d cluster network:
`docker network connect k3d-k3s-default registry.localhost`. And then you can
[check your local registry](#testing).

### <a name="etc-hosts"></a>Pushing to your local registry address

The registry will be located, by default, at `registry.localhost:5000` (customizable with the `--registry-name`
and `--registry-port` parameters). All the nodes in your k3d cluster can resolve this hostname (thanks to the
DNS server provided by the Docker daemon) but, in order to be able to push to this registry, this hostname
but also be resolved from your host.

Luckily (for Linux users), [NSS-myhostname](http://man7.org/linux/man-pages/man8/nss-myhostname.8.html) ships with many Linux distributions
and should resolve `*.localhost` automatically to `127.0.0.1`.  
Otherwise, it's installable using `sudo apt install libnss-myhostname`.

If it's not the case, you can add an entry in your `/etc/hosts` file like this:

```shell script
127.0.0.1 registry.localhost
```

Once again, this will only work with k3s >= v0.10.0 (see the [section below](#k3s-old)
when using k3s <= v0.9.1)

### <a name="registry-volume"></a>Local registry volume

The local k3d registry uses a volume for storying the images. This volume will be destroyed
when the k3d registry is released. In order to persist this volume and make these images survive
the removal of the registry, you can specify a volume with the `--registry-volume` and use the
`--keep-registry-volume` flag when deleting the cluster. This will create a volume with the given
name the first time the registry is used, while successive invocations will just mount this
existing volume in the k3d registry container.

### <a name="docker-hub-cache"></a>Docker Hub cache

The local k3d registry can also be used for caching images from the Docker Hub. You can start the
registry as a pull-through cache when the cluster is created with `--enable-registry-cache`. Used
in conjuction with `--registry-volume`/`--keep-registry-volume` can speed up all the downloads
from the Hub by keeping a persistent cache of images in your local machine.

## <a name="testing"></a>Testing your registry

You should test that you can

* push to your registry from your local development machine.
* use images from that registry in `Deployments` in your k3d cluster.

We will verify these two things for a local registry (located at `registry.localhost:5000`) running
in your development machine. Things would be basically the same for checking an external
registry, but some additional configuration could be necessary in your local machine when
using an authenticated or secure registry (please refer to Docker's documentation for this).

Firstly, we can download some image (like `nginx`) and push it to our local registry with:

```shell script
docker pull nginx:latest
docker tag nginx:latest registry.localhost:5000/nginx:latest
docker push registry.localhost:5000/nginx:latest
```

Then we can deploy a pod referencing this image to your cluster:

```shell script
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-test-registry
  labels:
    app: nginx-test-registry
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx-test-registry
  template:
    metadata:
      labels:
        app: nginx-test-registry
    spec:
      containers:
      - name: nginx-test-registry
        image: registry.localhost:5000/nginx:latest
        ports:
        - containerPort: 80
EOF
```

Then you should check that the pod is running with `kubectl get pods -l "app=nginx-test-registry"`.

## <a name="k3s-old"></a>Configuring registries for k3s <= v0.9.1

k3s servers below v0.9.1 do not recognize the `registries.yaml` file as we described in
the [previous section](#registries-file), so you will need to embed the contents of that
file in a `containerd` configuration file. You will have to create your own `containerd`
configuration file at some well-known path like `${HOME}/.k3d/config.toml.tmpl`, like this:

<pre>
# Original section: no changes
[plugins.opt]
path = "{{ .NodeConfig.Containerd.Opt }}"
[plugins.cri]
stream_server_address = "{{ .NodeConfig.AgentConfig.NodeName }}"
stream_server_port = "10010"
{{- if .IsRunningInUserNS }}
disable_cgroup = true
disable_apparmor = true
restrict_oom_score_adj = true
{{ end -}}
{{- if .NodeConfig.AgentConfig.PauseImage }}
sandbox_image = "{{ .NodeConfig.AgentConfig.PauseImage }}"
{{ end -}}
{{- if not .NodeConfig.NoFlannel }}
  [plugins.cri.cni]
    bin_dir = "{{ .NodeConfig.AgentConfig.CNIBinDir }}"
    conf_dir = "{{ .NodeConfig.AgentConfig.CNIConfDir }}"
{{ end -}}

# Added section: additional registries and the endpoints
[plugins.cri.registry.mirrors]
  [plugins.cri.registry.mirrors."<b>registry.localhost:5000</b>"]
    endpoint = ["http://<b>registry.localhost:5000</b>"]
</pre>

and then mount it at `/var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl` (where
the `containerd` in your k3d nodes will load it) when creating the k3d cluster:

```bash
k3d create \
    --volume ${HOME}/.k3d/config.toml.tmpl:/var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl
```
