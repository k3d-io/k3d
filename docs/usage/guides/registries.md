# Registries

## Registries configuration file

You can add registries by specifying them in a `registries.yaml` and mounting them at creation time:
`#!bash k3d create cluster mycluster --volume /home/YOU/my-registries.yaml:/etc/rancher/k3s/registries.yaml`.

This file is a regular [k3s registries configuration file](https://rancher.com/docs/k3s/latest/en/installation/private-registry/), and looks like this:

```yaml
mirrors:
  "my.company.registry:5000":
    endpoint:
      - http://my.company.registry:5000
```

In this example, an image with a name like `my.company.registry:5000/nginx:latest` would be
_pulled_ from the registry running at `http://my.company.registry:5000`.

Note well there is an important limitation: **this configuration file will only work with k3s >= v0.10.0**. It will fail silently with previous versions of k3s, but you find in the [section below](#k3s-old) an alternative solution.

This file can also be used for providing additional information necessary for accessing some registries, like [authentication](#authenticated-registries) and [certificates](#secure-registries).

### Authenticated registries

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

### Secure registries

When using secure registries, the [`registries.yaml` file](#registries-file) must include information about the certificates. For example, if you want to use images from the secure registry running at `https://my.company.registry`, you must first download a CA file valid for that server and store it in some well-known directory like `${HOME}/.k3d/my-company-root.pem`.  

Then you have to mount the CA file in some directory in the nodes in the cluster and include that mounted file in a `configs` section in the [`registries.yaml` file](#registries-file).
For example, if we mount the CA file in `/etc/ssl/certs/my-company-root.pem`, the `registries.yaml` will look like:

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

Finally, we can create the cluster, mounting the CA file in the path we specified in `ca_file`:

`#!bash k3d create cluster --volume ${HOME}/.k3d/my-registries.yaml:/etc/rancher/k3s/registries.yaml --volume ${HOME}/.k3d/my-company-root.pem:/etc/ssl/certs/my-company-root.pem`

## Using a local registry

### Using the k3d registry

!!! info "Not ported yet"
      The k3d-managed registry has not yet been ported from v1.x to v3.x

### Using your own local registry

You can start your own local registry it with some `docker` commands, like:

```bash
docker volume create local_registry
docker container run -d --name registry.localhost -v local_registry:/var/lib/registry --restart always -p 5000:5000 registry:2
```

These commands will start your registry in `registry.localhost:5000`. In order to push to this registry, you will need to make it accessible as described in the next section.
Once your registry is up and running, we will need to add it to your `registries.yaml` configuration file.
Finally, you have to connect the registry network to the k3d cluster network: `#!bash docker network connect k3d-k3s-default registry.localhost`. And then you can [test your local registry](#testing-your-registry).

### Pushing to your local registry address

As per the guide above, the registry will be available at `registry.localhost:5000`. All the nodes in your k3d cluster can resolve this hostname (thanks to the DNS server provided by the Docker daemon) but, in order to be able to push to this registry, this hostname also has to be resolved by your host.

Luckily (for Linux users), [NSS-myhostname](http://man7.org/linux/man-pages/man8/nss-myhostname.8.html) ships with many Linux distributions
and should resolve `*.localhost` automatically to `127.0.0.1`.  
Otherwise, it's installable using `sudo apt install libnss-myhostname`.

If it's not the case, you can add an entry in your `/etc/hosts` file like this:

```bash
127.0.0.1 registry.localhost
```

Once again, this will only work with k3s >= v0.10.0 (see the some sections below when using k3s <= v0.9.1)

## Testing your registry

You should test that you can

* push to your registry from your local development machine.
* use images from that registry in `Deployments` in your k3d cluster.

We will verify these two things for a local registry (located at `registry.localhost:5000`) running in your development machine. Things would be basically the same for checking an external registry, but some additional configuration could be necessary in your local machine when using an authenticated or secure registry (please refer to Docker's documentation for this).

First, we can download some image (like `nginx`) and push it to our local registry with:

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

## Configuring registries for k3s <= v0.9.1

k3s servers below v0.9.1 do not recognize the `registries.yaml` file as described in
the in the beginning, so you will need to embed the contents of that file in a `containerd` configuration file.
You will have to create your own `containerd` configuration file at some well-known path like `${HOME}/.k3d/config.toml.tmpl`, like this:

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

and then mount it at `/var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl` (where `containerd` in your k3d nodes will load it) when creating the k3d cluster:

```bash
k3d create cluster mycluster \
    --volume ${HOME}/.k3d/config.toml.tmpl:/var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl
```
