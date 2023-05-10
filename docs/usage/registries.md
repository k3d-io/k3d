# Using Image Registries

## Registries configuration file

You can add registries by specifying them in a `registries.yaml` and referencing it at creation time:
`#!bash k3d cluster create mycluster --registry-config "/home/YOU/my-registries.yaml"`.

This file is a regular [k3s registries configuration file](https://rancher.com/docs/k3s/latest/en/installation/private-registry/), and looks like this:

```yaml
mirrors:
  "my.company.registry:5000":
    endpoint:
      - http://my.company.registry:5000
```

In this example, an image with a name like `my.company.registry:5000/nginx:latest` would be _pulled_ from the registry running at `http://my.company.registry:5000`.

This file can also be used for providing additional information necessary for accessing some registries, like [authentication](#authenticated-registries) and [certificates](#secure-registries).

### Registries Configuration File embedded in k3d's SimpleConfig

If you're using a `SimpleConfig` file to configure your k3d cluster, you may as well embed the registries.yaml in there directly:

```yaml
apiVersion: k3d.io/v1alpha5
kind: Simple
metadata:
  name: test
servers: 1
agents: 2
registries:
  create: 
    name: myregistry
  config: |
    mirrors:
      "my.company.registry":
        endpoint:
          - http://my.company.registry:5000
```

Here, the config for the k3d-managed registry, created by the `create: {...}` option will be merged with the config specified under `config: |`.

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

```bash
k3d cluster create \
  --volume "${HOME}/.k3d/my-registries.yaml:/etc/rancher/k3s/registries.yaml" \
  --volume "${HOME}/.k3d/my-company-root.pem:/etc/ssl/certs/my-company-root.pem"
```

## Using a local registry

### Preface: Referencing local registries

In the next sections, you're going to create a local registry (i.e. a container image registry running in a container in your docker host).  
That container will have a name, e.g. `mycluster-registry`.  
If you follow the guide closely (or definitely if you use the k3d-managed option), this name will be known to all the hosts (K3s containers) and workloads in your k3d cluster.  
However, you usually want to push images into that registry from your local machine, which **does not know** that name by default.  
Now you have a few options, including the following three:  

1. Use `localhost`: Since the container will have a port mapped to your local host, you can just directly reference it via e.g. `localhost:12345`, where `12345` is the mapped port
   - If you later pull the image from the registry, only the repository path (e.g. `myrepo/myimage:mytag` in `mycluster-registry:5000/myrepo/myimage:mytag`) matters to find your image in the targeted registry.
2. Get your machine to know the container name: For this you can use the plain old hosts file (`/etc/hosts` on Unix systems and `C:\windows\system32\drivers\etc\hosts` on Windows) by adding an entry like the following to the end of the file:  

  ```text
  127.0.0.1 mycluster-registry
  ```

3. Use some special resolving magic: Tools like `dnsmasq` or `nss-myhostname` (see info box below) and others can setup your local resolver to directly resolve the registry name to `127.0.0.1`.

!!! info "nss-myhostname to resolve `*.localhost`"
    Luckily (for Linux users), [NSS-myhostname](http://man7.org/linux/man-pages/man8/nss-myhostname.8.html) ships with many Linux distributions
    and should resolve `*.localhost` automatically to `127.0.0.1`.  
    Otherwise, it's installable using `sudo apt install libnss-myhostname`.

### Using k3d-managed registries

#### Create a dedicated registry together with your cluster

1. `#!bash k3d cluster create mycluster --registry-create mycluster-registry`: This creates your cluster `mycluster` together with a registry container called `mycluster-registry`

  - k3d sets everything up in the cluster for containerd to be able to pull images from that registry (using the `registries.yaml` file)
  - the port, which the registry is listening on will be mapped to a random port on your host system

2. Check the k3d command output or `#!bash docker ps -f name=mycluster-registry` to find the exposed port
3. [Test your registry](#testing-your-registry)

#### Create a customized k3d-managed registry

1. `#!bash k3d registry create myregistry.localhost --port 12345` creates a new registry called `k3d-myregistry.localhost` (could be used with automatic resolution of `*.localhost`, see next section - also, **note the `k3d-` prefix** that k3d adds to all resources it creates)
2. `#!bash k3d cluster create newcluster --registry-use k3d-myregistry.localhost:12345` (make sure you use the **`k3d-` prefix** here) creates a new cluster set up to use that registry
3. [Test your registry](#testing-your-registry)

### Using your own (not k3d-managed) local registry

_We recommend using a k3d-managed registry, as it plays nicely together with k3d clusters, but here's also a guide to create your own (not k3d-managed) registry, if you need features or customizations, that k3d does not provide:_

??? nonk3dregistry "Using your own (not k3d-managed) local registry"

    You can start your own local registry it with some `docker` commands, like:

    ```bash
    docker volume create local_registry
    docker container run -d --name registry.localhost -v local_registry:/var/lib/registry --restart always -p 12345:5000 registry:2
    ```

    These commands will start your registry container with name and port (on your host) `registry.localhost:12345`. In order to push to this registry, you will need to make it accessible as described in the next section.  
    Once your registry is up and running, we will need to add it to your `registries.yaml` configuration file.  
    Finally, you have to connect the registry network to the k3d cluster network: `#!bash docker network connect k3d-k3s-default registry.localhost`.  
    And then you can [test your local registry](#testing-your-registry).

### Pushing to your local registry address

!!! info "See Preface"
    The information below has been addressed in the [preface for this section](#preface-referencing-local-registries).

## Testing your registry

You should test that you can

- push to your registry from your local development machine.
- use images from that registry in `Deployments` in your k3d cluster.

We will verify these two things for a local registry (located at `k3d-registry.localhost:12345`) running in your development machine.  
Things would be basically the same for checking an external registry, but some additional configuration could be necessary in your local machine when using an authenticated or secure registry (please refer to Docker's documentation for this).

**Assumptions**: In the following test cases, we assume that the registry name `k3d-registry.localhost` resolves to `127.0.0.1` in your local machine (see [section preface for more details](#preface-referencing-local-registries)) and to the registry container IP for the k3d cluster nodes (K3s containers).

**Note**: as per the explanation in the [preface](#preface-referencing-local-registries), you could replace `k3d-registry.localhost:12345` with `localhost:12345` in the `docker tag` and `docker push` commands below (but not in the `kubectl` part!)

### Nginx Deployment

First, we can download some image (like `nginx`) and push it to our local registry with:

```bash
docker pull nginx:latest
docker tag nginx:latest k3d-registry.localhost:12345/nginx:latest
docker push k3d-registry.localhost:12345/nginx:latest
```

Then we can deploy a pod referencing this image to your cluster:

```bash
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
        image: k3d-registry.localhost:12345/nginx:latest
        ports:
        - containerPort: 80
EOF
```

Then you should check that the pod is running with `kubectl get pods -l "app=nginx-test-registry"`.

### Alpine Pod

1. Pull the alpine image: `#!bash docker pull alpine:latest`
2. re-tag it to reference your newly created registry: `#!bash docker tag alpine:latest k3d-registry.localhost:12345/testimage:local`
3. push it: `#!bash docker push k3d-registry.localhost:12345/testimage:local`
4. Use kubectl to create a new pod in your cluster using that image to see, if the cluster can pull from the new registry: `#!bash kubectl run --image k3d-registry.localhost:12345/testimage:local testimage --command -- tail -f /dev/null`
   - (creates a container that will not do anything but keep on running)

## Creating a registry proxy / pull-through registry

1. Create a pull-through registry

    ```bash
    k3d registry create docker-io `# Create a registry named k3d-docker-io` \
      -p 5000 `# listening on local host port 5000` \ 
      --proxy-remote-url https://registry-1.docker.io `# let it mirror the Docker Hub registry` \
      -v ~/.local/share/docker-io-registry:/var/lib/registry `# also persist the downloaded images on the device outside the container`
    ```

2. Create `registry.yml`

    ```yaml
    mirrors:
      "docker.io":
        endpoint:
          - http://k3d-docker-io:5000
    ```

3. Create a cluster and using the pull-through cache

    ```bash
    k3d cluster create cluster01 --registry-use k3d-docker-io:5000 --registry-config registry.yml
    ```

4. After cluster01 ready, create another cluster with the same registry or rebuild the cluster, it will use the already locally cached images.

    ```bash
    k3d cluster create cluster02 --registry-use k3d-docker-io:5000 --registry-config registry.yml
    ```

### Creating a registry proxy / pull-through registry via configfile

1. Create a config file, e.g. `/home/me/test-regcache.yaml`

    ```yaml
    apiVersion: k3d.io/v1alpha5
    kind: Simple
    metadata:
      name: test-regcache
    registries:
      create:
        name: docker-io # name of the registry container
        proxy:
          remoteURL: https://registry-1.docker.io # proxy DockerHub
        volumes:
          - /tmp/reg:/var/lib/registry # persist data locally in /tmp/reg
      config: | # tell K3s to use this registry when pulling from DockerHub
        mirrors:
          "docker.io":
            endpoint:
              - http://docker-io:5000
    ```

2. Create cluster from config:

    ```bash
    k3d cluster create -c /home/me/test-regcache.yaml
    ```
