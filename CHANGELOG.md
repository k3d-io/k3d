# Changelog

## v5.7.0 - 04.07.2024

### Added

- feat: support config embedded and external files (#1417)
- docs: add examples for config embedded and external files (#1432)
- feat: compatibility with docker userns-remap (#1442) 
- docs: mention ipam when creating multiserver cluster (#1451) 

### Changed

- docs: Update CUDA docs to use k3s suggested method (#1430)
- chore: upgrade go + dependencies + address all golangci-lint issues + fix deprecations (#1459)
- chore: upgrade docker dependency and adjust for deprecations (#1460) 

### Fixed

- fix: close output file (#1436)
- fix: Script exits fatally when resolv.conf is missing Docker nameserver (#1441)
- test: fix translate.go test following userns merge (#1444) 
- fix: respect ~/.kube/config as a symlink (#1455)
- fix: preserve coredns config during cluster restart (#1453)
  - **IMPORTANT** This makes use of the `coredns-custom` configmap, so please consider this in case you're using this configmap yourself!
- fix: make drain ignore DaemonSets & bypass PodDisruptionBudgets (#1414) 

## v5.6.3 - 10.04.2024

### Changed

- Dependency updates and related fixes

## v5.6.2 - 09.04.2024

### Added

-  [DOCS] Add scoop install option (#1390)
- feat: support writing kubeconfig to a stream (#1381)

### Changed

- Not using stdout directly for logging (#1339)
- change: enable fixes by default and consolidate lookup logic (#1349)
- Consistent logging during cluster creation flow (#1398)
- 

### Fixed

- change: fix docs link (#1343)
- 

## v5.6.0 - 21.08.2023

### Added

- add: iptables in DinD image (#1298)
- docs(podman): add usage for rootless mode on macOS (#1314)

### Changed

- **Potentially Breaking**: For people using k3d as a module: switch from netaddr.af to netipx + netip (changed some code around `host.k3d.internal` and the docker runtime)
- **Potentially Breaking**: K3d config directory may change for you: Adhere to XDG's configuration specification (#1320)

### Fixed

- docs: fix go install command (#1337)
- fix docs links in CONTRIBUTING.md
- chore: pkg imported more than once (#1313)

## v5.5.2 - 03.08.2023

### Fixed

- docs: fix list failing to render (#1300)
- bump dependencies to fix `Invalid Host Header` issue with [Docker/Moby#45935](https://github.com/moby/moby/issues/45935)

### Changed

- change: proxy - update nginx-alpine base image (#1309)
- change: add empty /tmp to binary-only image to make it work with config files

### Added

- add: workflow to label issues/prs by sponsors

## v5.5.1 - 19.05.2023

### Fixed

- fix/regression: custom registry config not parsed correctly (#1292)

## v5.5.0 - 17.05.2023

### Added

- Add support for ulimits (#1264)
  - new flag: `k3d cluster create --runtime-ulimit NAME[=SOFT]:[HARD]` (same for `k3d node create`)
- add: K3D_FIX_MOUNTS fix to make / rshared (e.g. to make Cilium work) (#1268)
  - new environment variable: `K3D_FIX_MOUNTS=1`
- add(docs): podman instructions for macOS (#1257)
- Adds json response of version info (#1262)
  - new flag: `k3d version -o json`

### Changed

- change: allow full K3s registry configuration (#1215)
- change: update deps (manual + dependabot)
- change: set e2e test ghaction timeout
- change: improved help text for k3d version ls
- change: deprecate 'k3d version ls --format' in favor of '--output'
- change: golangci-lint fix whitespaces
- change: udpate docs

### Fixed

- Fix panic when k3sURLEnvIndex is -1 (#1252)
- Fix spelling mistake in configfile.md (#1261)
- Correct typo: Inconsistent filename in registry documentation. (#1275)
- fix: k3d version ls (now via crane) (#1286)
- fix: registries.yaml file not marshalled correctly by k8s yaml package

### Deprecated

- change: deprecate 'k3d version ls --format' in favor of '--output'

## v5.4.9 - 16.03.2023 [BROKEN BUILD]

### Changed

- Updated docker dependency to v23.0.1
- change: replace deprecated set-output command with environment file in Github Actions (#1226)

### Fixed

- fix: go install was failing due to outdated docker dependency
- fix: handle colima host for host.k3d.internal lookup (#1228)

## v5.4.8 - 04.03.2023

### Changed

- Go 1.20 and updated dependencies
- change: Use loadbalancer or any *active* server as K3S_URL (#1190)
- change: graceful shutdown drains node before k3d container stops (#1119)
- change: update docs to use quotes around extra args (#1218)
- changed: update podman service documentation around network dns (#1210)
- change: no whitespace in goflags in makefile
- change: fix build with go 1.20 (#1216)

### Fixed

- fix: generate checksum for k3d binaries (#1209)
- fix: improved error handling when update.k3s.io returns a 5XX or invalid response (#1170)
- fix: install script on windows (#1168)
- fix: fix for link in doc (#1219)

## v5.4.7 - 02.02.2023

### Changed

- updated direct and transitive dependencies

### Fixed

- fix: avoid appending existing volumes (#1154)
- fix: indentation for CoreDNS doc (#1166)
- fix: logs error shadowing exec error (#1172)
- docs: Add missing backtick to k3s-arg example command (#1192)
- Support reading in registries-config via env (#1199)

## v5.4.6 - 29.08.2022

### Added

- add: ability to load configuration from stdin (#1126)

### Changed

- update dependencies
- introduce Go workspace mode
- updated docker/k3s version test-matrix
- Go 1.19
- More info on "node stopped returning log lines" error

### Fixed

- tests/e2e: failing e2e tests for parsing config file from stdin
- ci: "random" failing GitHub Actions due to "too many open files"
- docs: fix code highlighting
- docs: beautify bash commands (#1103)

## v5.4.5 - Broken/Unreleased

- This tag was reverted because of constant failures in GitHub Actions and the E2E Tests

## v5.4.4 - 11.07.2022

### Added

- Docs: Clarification of Network Policies in K3s (#1081)

### Changed

- Sponsorship information and updated issue templates
- Switch to `sigs.k8s.io/yaml` everywhere in the project to allow for consistent json/yaml output (#1094)

### Fixed

- Support running k3d with podman in rootless mode using cgroups v2 (#1084)
- `k3d config init` used the legacy config format (#1091)
- Properly handle image prefix "docker.io", etc during image import (#1096)

## v5.4.3 - 07.06.2022

### Added

- Support for pull-through registry (#1075)
  - In command `k3d registry create`
    - e.g. `k3d registry create --proxy-remote-url https://registry-1.docker.io -p 5000 -v /tmp/registry:/var/lib/registry`
  - In config file:

      ```yaml
      # ...
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

  - See registry documentation

## v5.4.2 - 04.06.2022

### Added

- Docs: `hostAliases` in the config file
- New field `registries.create.image` (same as `k3d registry create --image`) in config `v1alpha4` (no version bump) (#1056)

### Changed

- Go 1.18

### Fixed

- docs: fix defaults-networking href (#1064)
- fix deleting of cluster by config file (#1054)
- fix: DOCKER_HOST handling of unix sockets (#1045)
- make: Use go install instead of go get for installing tools (#1038)
- fix: e2e tests safe git directory

## v5.4.1 - 29.03.2022

### Changed

- Updated dependencies (docker, containerd, etc.)

## v5.4.0 - 26.03.2022

**Note**: This is the **first independent release** of k3d

  - k3d moved from rancher/k3d to k3d-io/k3d
  - k3d is fully community-owned
  - k3d does not depend on any company's toolchain or accounts

**Note 2**: You can now fund the work on k3d using GitHub Sponsors ([@iwilltry42](https://github.com/sponsors/iwilltry42)) or IssueHunt ([k3d-io/k3d](https://issuehunt.io/r/k3d-io/k3d))

### Added

- GitHub Actions Release Workflow (#977 & #1024)
  - Replaces DroneCI
  - Now uses `buildx` & `buildx bake` for multiplatform builds (instead of VMs with the according architectures)
  - Now pushes to GHCR instead of DockerHub
- docs: added FAQ entry on using Longhorn in k3d
- docs: added config file tip that k3d expands environment variables
- docs: added section about using k3d with Podman (#987)
- docs: add connect section on homepage (#988)
- added `k3d node create --k3s-arg` flag (#1032)

### Changed

- references to rancher/k3d updated to k3d-io/k3d (#976)
- reference to rancher/k3s updated to k3s-io/k3s (#985)
- explicitly set `bridge` mode for k3d-created networks for Podman compatibility (#986)
- use secure defaults for curl in install script (#999)
- chore: update docs requirements and re-run docgen for commands (#1033)
- change: no default image for node creation in local cluster where image should be copied from existing nodes (#1034)

### Fixed

- fixed volume shortcuts not working because clusterconfig was not being processed
- fixed AUR Release pipeline with more relaxed version selection (#966)
- fixed ZSH completion output (#1014)
- Do not defer goroutine to delete tools node, as this leads to errors
- Hotfix: switch default for image import to original tools-node mode, as the new direct mode fails fairly often
- GetGatewayIP for host.k3d.internal should error out if there's no gateway defined (#1027)
- Store hostAliases in label to persist them across cluster stop/start (#1029)

### Deprecated

- DockerHub Images: k3d's images will now be pushed to GHCR under <https://github.com/orgs/k3d-io/packages?repo_name=k3d>

### Removed

- DroneCI Test & Release Pipeline

### Compatibility

This release was [automatically tested](https://github.com/k3d-io/k3d/actions/runs/2044325827) with the following setups:

#### Docker

- 20.10.5
- 20.10.12

**Expected to Fail** with the following versions:

- <= 20.10.4 (due to runc, see <https://github.com/rancher/k3d/issues/807>)

#### K3s

We test a full cluster lifecycle with different [K3s channels](https://update.k3s.io/v1-release/channels), meaning that the following list refers to the current latest version released under the given channel:

- Channel v1.23
- Channel v1.22

**Expected to Fail** with the following versions:

- <= v1.18 (due to not included, but expected CoreDNS in K3s)

## v5.3.0 - 03.02.2022

**Note:** Now trying to follow a standard scheme defined by <https://keepachangelog.com/en/1.0.0/>

### Added

- new config options to configure extra hosts by @iwilltry42 in <https://github.com/rancher/k3d/pull/938>
- host pid mode support for k3s-server and k3s-agent by @hlts2 in <https://github.com/rancher/k3d/pull/929>
- SimpleConfig v1alpha4 by @iwilltry42 in <https://github.com/rancher/k3d/pull/944>
- add env var LOG_COLORS=[1|true|0|false] to toggle colored log output (enabled by default) by @iwilltry42 in <https://github.com/rancher/k3d/pull/951>
- Compatibility Tests by @iwilltry42 in <https://github.com/rancher/k3d/pull/956>
- Volume Shortcuts and k3d-managed volumes by @iwilltry42 in <https://github.com/rancher/k3d/pull/916>
  - Use some destination shortcuts with the `--volume/-v` flag that k3d automatically expands
    - `k3s-storage` -> `/var/lib/rancher/k3s/storage`
    - `k3s-manifests` -> `/var/lib/rancher/k3s/server/manifests`
    - `k3s-manifests-custom` -> `/var/lib/rancher/k3s/server/manifests/custom` (not K3s default: this is just some sub-directory inside the auto-deploy manifests directory which will also be parsed)
    - `k3s-containerd` -> `/var/lib/rancher/k3s/agent/etc/containerd/config.toml` (use with caution, K3s generates this file!)
    - `k3s-containerd-tmpl` -> `/var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl` (used by K3s to generate the real config above)
    - `k3s-registry-config` -> `/etc/rancher/k3s/registries.yaml` (or just use `--registry-config`)
  - k3d-managed volumes
    - non-existing named volumes starting with a `k3d-` prefix will now be created and managed by `k3d`
- JSON schema versions in-repo to link to from schemastore.org by @iwilltry42 in <https://github.com/rancher/k3d/pull/942>

### Changed

- Config file compatible with Kustomize by @erikgb in <https://github.com/rancher/k3d/pull/945>
- chore: update direct dependencies by @iwilltry42 in <https://github.com/rancher/k3d/pull/935>

### Fixed

- serverlb should be created before using and restarted unless stopped by @wymli in <https://github.com/rancher/k3d/pull/948>
- fix typo in node.go by @eltociear in <https://github.com/rancher/k3d/pull/949>
- mutex on ensureToolsNode to avoid duplicate container name causing error by @iwilltry42 in <https://github.com/rancher/k3d/pull/952>
- detect '--disable=coredns' and conditionally disable injection by @iwilltry42 in <https://github.com/rancher/k3d/pull/955>
- invert logic for LOG_LEVEL parsing by @myitcv in <https://github.com/rancher/k3d/pull/958>

### Deprecated

- SimpleConfig API version `k3d.io/v1alpha3` is now deprecated in favor of `k3d.io/v1alpha4`

### Removed

- unused volume validation functionality in `cmd/util`, does not affect the CLI (#916)

### Compatibility

This release was automatically tested with the following setups:

#### Docker

- 20.10.5
- 20.10.12

**Expected to Fail** with the following versions:

- <= 20.10.4 (due to runc, see <https://github.com/rancher/k3d/issues/807>)

#### K3s

We test a full cluster lifecycle with different [K3s channels](https://update.k3s.io/v1-release/channels), meaning that the following list refers to the current latest version released under the given channel:

- Channel v1.23
- Channel v1.22

**Expected to Fail** with the following versions:

- <= v1.18 (due to not included, but expected CoreDNS in K3s)

## v5.2.2

### Fixes

- mitigate issue when importing images from multiple tars (#881, @sbaier1)
- fix: cluster delete should not fail if no cluster was found by config file (#886, @kuritka)

### Misc

- docs: new page about k3d concepts, incl. nodefilters (#888)
  - <https://k3d.io/v5.2.1/design/concepts/#nodefilters>

## v5.2.1

### Features & Enhancements

- improved Podman compatibility (#868, @serverwentdown)
  - last missing piece: release of <https://github.com/containers/podman/pull/12328>
- improved error handling and logs when waiting for container logs (ca47fac)

### Fixes

- fix: only replace default api host with docker host (#879)
- fix: use available hardcoded K3s version in version.go (0bbb5b9)

## v5.2.0

### Features & Enhancements

- Improve image import performance (#826, @sbaier1)
  - **New flag**: `k3d image import --mode [auto | direct | tools]`
    - `tools` is the old default, which spawns a `k3d-tools` container for importing
    - `auto` is the new default to automatically detect which mode should work best
    - `direct` directly streams the images into the node containers without the `k3d-tools` container
- Enhanced usability of nodefilters & error messages for wrong usage (#871)
- **New command**: `k3d version list [k3s | k3d | k3d-proxy | k3d-tools]` to get image tags that can be used with k3d (#870)
  - e.g. use `k3d version list k3s --format repo` to get the latest image available for K3s and use it via `k3d cluster create --image <image>`
  - Docs: [docs/usage/commands/k3d_version_list.md](./docs/usage/commands/k3d_version_list.md)

### Fixes

- cluster network: reserve IP extra IP for k3d-tools container in k3d-managed IPAM to avoid conflicts
- process the SimpleConfig before validating it to avoid early exit in hostnetwork mode (#860)
- error out if `K3D_FIX_DNS=1` is set and user tries to mount a file to `/etc/resolv.conf` (conflict)
- clusterStart: only run actions which are necessary given the start reason (e.g. `cluster start` vs. `cluster create`)
- fix injection of `host.k3d.internal` based on resolving `host.docker.internal` (#872)
  - also now uses `host.docker.internal` in kubeconfig based on certain conditions (see PR)

### Misc

- tests/e2e: parellelize and cleanup tests -> cut execution speed in half (#848 & #849)
  - also run some make targets in parallel
  - new env var `E2E_PARALLEL=<int>` to configure parallelism
  - test output is now redirected to files inside the runner and only the logs of failed tests will later be output
- Update dependencies, including docker, containerd & k8s
- docs: clarify usage of local registries with k3d
- docs: fix port numbers in registry usage guide

### Notes

- k3d v5.x.x requires at least docker version 20.10.4

## v5.1.0

### Features

- clusterCreate: `--image` option (also in config file) magic words to follow K3s channels (#841)
  - `latest`/`stable` to follow latest/stable channels of K3s
  - `+<channel>` (prefix `+`) where `<channel>` can as well be `latest` or `stable`, but also e.g. `v1.21`
  - k3d will then check the K3s channel server to get the latest image for that channel

### Enhancements

- nodeHooks: add descriptions and log them for more verbosity (#843)
- `node create`: inject `host.k3d.internal` into `/etc/hosts` similar to the `cluster create` command (#843)

### Fix

- `--network host`: do not do any network magic (like `host.k3d.internal` injection, etc.) when `host` network is used (#844)

### Misc

- CI/Makefile: build with `-mod vendor`
- docs: document using some K3s features in k3d, including `servicelb`, `traefik`, `local-storage-provisioner` and `coredns` (#845)

## v5.0.3

### Enhancements & Fixes

- simplified way of getting a Docker API Client that works with Docker Contexts and `DOCKER_*` environment variable configuration (#829, @dragonflylee)
  - fix: didn't honor `DOCKER_TLS` environment variables before

## v5.0.2

### Enhancements

- CoreDNS Configmap is now edited in the auto-deploy manifest on disk instead of relying on `kubectl patch` command (#814)
- refactor: add cmd subcommands in a single function call (#819, @moeryomenko)
- handle ready-log-messages by type and intent & check them in single log streams instead of checking whole chunks every time (#818)

### Fixes

- fix: config file check failing with env var expansion because unexpanded input file was checked

### Misc

- cleanup: ensure that connections/streams are closed once unused (#818)
- cleanup: split type definitions across multiple files to increase readability (#818)
- docs: clarify `node create` help text about cluster reference (#808, @losinggeneration)
- refactor: move from io/ioutil (deprecated) to io and os packages (#827, @Juneezee)

## v5.0.1

### Enhancement

- add `HostFromClusterNetwork` field to `LocalRegistryHosting` configmap as per KEP-1755 (#754)

### Fixes

- fix: nilpointer exception on failed exec process with no returned logreader
- make post-create cluster preparation (DNS stuff mostly) more resilient (#780)
- fix v1alpha2 -> v1alpha3 config migration (and other related issues) (#799)

### Misc

- docs: fix typo (#784)
- docs: fix usage of legacy `--k3s-agent/server-arg` flag

## v5.0.0

This release contains a whole lot of new features, breaking changes as well as smaller fixes and improvements.
The changelog shown here is likely not complete but gives a broad overview over the changes.
For more details, please check the v5 milestone (<https://github.com/rancher/k3d/milestone/27>) or even the commit history.
The docs have been updated, so you should also find the information you need there, with more to come!

The demo repository has also been updated to work with k3d v5: <https://github.com/iwilltry42/k3d-demo>.

**Info**: <https://k3d.io> is now versioned, so you can checkout different versions of the documentation by using the dropdown menu in the page title bar!

**Feedback welcome!**

### Breaking Changes

- new syntax for nodefilters
  - dropped the usage of square brackets `[]` for indexing, as it caused problems with some shells trying to interpret them
  - new syntax: `@identifier[:index][:opt]` (see <https://github.com/rancher/k3d/discussions/652>)
    - example for a port-mapping: `--port 8080:80@server:0:proxy`
      - identifier = `server`, index = `0`, opt = `proxy`
      - `opt` is an extra optional argument used for different purposes depending on the flag
        - currently, only the `--port` flag has `opt`s, namely `proxy` and `direct` (see other breaking change)
- port-mapping now go via the loadbalancer (serverlb) by default
  - the `--port` flag has the `proxy` opt (see new nodefilter syntax above) set by default
  - to leverage the old behavior of direct port-mappings, use the `direct` opt on the port flag
  - the nodefilter `loadbalancer` will now do the same as `servers:*;agents:*` (proxied via the loadbalancer)
- flag `--registries-create` transformed from bool flag to string flag: let's you define the name and port-binding of the newly created registry, e.g. `--registry-create myregistry.localhost:5001`

### Fixes

- cleaned up and properly sorted the sanitization of existing resources used to create new nodes (#638)

### Features & Enhancements

- new command: `k3d node edit` to edit existing nodes (#615)
  - currently only allows `k3d node edit NODE --port-add HOSTPORT:CONTAINERPORT` for the serverlb/loadbalancer to add new ports
  - pkg: new `NodeEdit` function
- new (hidden) command: `k3d debug` with some options for debugging k3d resources (#638)
  - e.g. `k3d debug loadbalancer get-config` to get the current loadbalancer configuration
- loadbalancer / k3d-proxy (#638)
  - updated fork of `confd` to make usage of the file backend including a file watcher for auto-reloads
    - this also checks the config before applying it, so the lb doesn't crash on a faulty config
  - updating the loadbalancer writes the new config file and also checks if everything's going fine afterwards
  - some settings of the loadbalancer can now be configured using `--lb-config-override`, see docs at <https://k3d.io/v5.0.0/design/defaults/#k3d-loadbalancer>
- helper images can now be set explicitly via environment variables: `K3D_IMAGE_LOADBALANCER` & `K3D_IMAGE_TOOLS` (#638)
- concurrently add new nodes to an existing cluster (remove some dumb code) (#640)
  - `--wait` is now the default for `k3d node create`
- normalized flag usage for k3s and runtime (#598, @ejose19)
  - rename `k3d cluster create --label` to `k3d cluster create --runtime-label` (as it's labelling the node on runtime level, e.g. docker)
    - config option moved to `options.runtime.labels`
  - add `k3d cluster create --k3s-node-label` to add Kubernetes node labels via k3s flag (#584, @developer-guy, @ejose, @dentrax)
    - new config option `options.k3s.nodeLabels`
  - the same for `k3d node create`
- improved config file handling (#605)
  - new version `v1alpha3`
    - warning when using outdated version
    - validation dynamically based on provided config apiVersion
    - new default for `k3d config init`
  - new command `k3d config migrate INPUT [OUTPUT]` to migrate config files between versions
    - currently supported migration `v1alpha2` -> `v1alpha3`
  - pkg: new `Config` interface type to support new generic `FromViper` config file parsing
- changed flags `--k3s-server-arg` & `--k3s-agent-arg` into `--k3s-arg` with nodefilter support (#605)
  - new config path `options.k3s.extraArgs`
- config file: environment variables (`$VAR`, `${VAR}` will be expanded unconditionally) (#643)
- docker context support (#601, @developer-guy & #674)
- Feature flag using the environment variable `K3D_FIX_DNS` and setting it to a true value (e.g. `export K3D_FIX_DNS=1`) to forward DNS queries to your local machine, e.g. to use your local company DNS

### Misc

- tests/e2e: timeouts everywhere to avoid killing DroneCI (#638)
- logs: really final output when creating/deleting nodes (so far, we were not outputting a final success message and the process was still doing stuff) (#640)
- tests/e2e: add tests for v1alpha2 to v1alpha3 migration
- docs: use v1alpha3 config version
- docs: update general appearance and cleanup

## v4.4.8

## Enhancements

- Improved DroneCI Pipeline for Multiarch Images and SemVer Tags (#712)
  - **Important**: New images will not have the `v` prefix in the tag anymore!
    - but now real releases will use the "hierarchical" SemVer tags, so you could e.g. subscribe to rancher/k3d-proxy:4 to get v4.x.x images for the proxy container

## Fixes

- clusterCreate: do not override hostIP if hostPort is missing (#693, @lukaszo)
- imageImport: import all listed images, not only the first one (#701, @mszostok)
- clusterCreate: when memory constraints are set, only pull the image used for checking the edac folder, if it's not present on the machine
- fix: update k3d-tools dependencies and use API Version Negotiation, so it still works with older versions of the Docker Engine (#679)

### Misc

- install script: add darwin/arm64 support (#676, @colelawrence)
- docs: fix go install command (#677, @Rots)
- docs: add project overview (<https://k3d.io/internals/project/>) (#680)

## v4.4.7

### Features / Enhancements

- new flag: `k3d image import --keep-tools` to not delete the tools node container after importing the image(s) (#672)
- improve image name handling when importing images (#653, @cimnine)
  - normalize image names internally, e.g. strip prefixes that docker adds, but that break the process
  - see <https://k3d.io/usage/commands/k3d_image_import/> for more info

### Fixes

- Use default gateway, when bridge network doesn't have it (#666, @kuritka)
- Start an existing, but not running tools node to re-use it when importing an image (#672)

### Misc

- deps: switching back to upstream viper including the StringArray fix
- docs: reference to "nolar/setup-k3d-k3s" step for GitHub Actions (#668, @nolar)
- docs: updated and simplified CUDA guide (#662, @vainkop) (#669)

## v4.4.6

### Fixes

- fix an issue where the cluster creation would stall waiting for the `starting worker processes` log message from the loadbalancer/serverlb
  - this was likely caused by a rounding issue when asking docker to get the container logs starting at a specific timestamp
  - we now drop subsecond precision for this to avoid the rounding issue, which was confirmed to work
  - see issues #592 & #621

### Misc

- to debug the issue mentioned above, we introduced a new environment variable `K3D_LOG_NODE_WAIT_LOGS`, which can be set to a list of node roles (e.g. `K3D_LOG_NODE_WAIT_LOGS=loadbalancer,agent`) to output the container logs that k3d inspects

## v4.4.5

### Fixes

- overall: use the getDockerClient helper function everywhere to e.g. support docker via ssh everywhere
- nodeCreate: do not copy meminfo/edac volume mounts from existing nodes, to avoid conflicts with generated mounts
- kubeconfig: fix file handling on windows (#626 + #628, @dragonflylee)

### Misc

- docs: add [FAQ entry](https://k3d.io/faq/faq/#nodes-fail-to-start-or-get-stuck-in-notready-state-with-log-nf_conntrack_max-permission-denied) on nf_conntrack_max: permission denied issue from kube-proxy (#607)
- docs: cleanup, fix formatting, etc.
- license: update to include 2021 in time range
- docs: link to AutoK3s (#614, @JacieChao)
- tests/e2e: update the list of tested k3s versions

## v4.4.4

### Enhancements

- nodes created via `k3d node create` now inherit the registry config from existing nodes (if there is any) (#597)
- the cgroupv2 hotfix (custom entrypoint script) is now enabled by default (#603)
  - disable by setting the environment variable `K3D_FIX_CGROUPV2=false`

### Fixes

- fix using networks without IPAM config (e.g. `host`)

### Misc

- docs: edit links on k3d.io now point to the correct branch (`main`)
- docs: new FAQ entry on spurious PID entries when using shared mounts (#609, @leelavg)

## v4.4.3

### Highlights

- cgroupv2 support: to properly work on cgroupv2 systems, k3s has to move all the processes from the root cgroup to a new /init cgroup and enable subtree_control
  - this is going to be included in the k3s agent code directly (<https://github.com/k3s-io/k3s/pull/3242>)
  - for now we're overriding the container entrypoint with a script that does this (#579, compare <https://github.com/k3s-io/k3s/pull/3237>)
  - thanks a lot for all the input and support @AkihiroSuda
  - **Usage**: set the environment variable `K3D_FIX_CGROUPV2` to a `true` value before/when creating a cluster with k3d
    - e.g. `export K3D_FIX_CGROUPV2=1`

### Fixes

- fix: docker volume not mountable due to validation failure
  - was not able to mount named volume on windows as we're checking for `:` meant for drive-letters and k3d separators

### Misc

- fix create command's flags typo (#568, @Jason-ZW)

## v4.4.2

### Fixes

- k3d-proxy: rename udp upstreams to avoid collisions/duplicates (#564)

### Features

- add *hidden* command `k3d runtime-info` used for debugging (#553)
  - this comes with some additions on package/runtime level
- add *experimental* `--subnet` flag to get some k3d IPAM to ensure that server nodes keep static IPs across restarts (#560)

### Misc

- docs: fix typo (#556, @gcalmettes)
- docs: fix typo (#561, @alechartung)
- ci/drone: pre-release on `-dev.X` tags
- ci/drone: always build no matter the branch name (just not release)
- docs: add automatic command tree generation via cobra (#562)
- makefile: use `go env gopath` as install target for tools (as per #445)
- JSONSchema: add some examples and defaults (now also available via <https://raw.githubusercontent.com/rancher/k3d/main/pkg/config/v1alpha2/schema.json> in your IDE)

## v4.4.1

### Fixes

- use viper fork that contains a fix to make cobra's `StringArray` flags work properly
  - this fixes the issue, that flag values containing commas got split (because we had to use `StringSlice` type flags)
  - this is to be changed back to upstream viper as soon as <https://github.com/spf13/viper/pull/398> (or a similar fix) got merged

## v4.4.0

### Features / Enhancements

- Support for Memory Limits using e.g. `--servers-memory 1g` or `--agents-memory 1.5g` (#494, @konradmalik)
  - enabled by providing fake `meminfo` files

### Fixes

- fix absolute paths in volume mounts on Windows (#510, @markrexwinkel)

### Documentation

- clarify registry names in docs and help text
- add usage section about config file (#534)
- add FAQ entry on certificate error when running behind corporate proxy
- add MacPorts install instructions (#539, @herbygillot)
- Heal Shruggie: Replace amputated arm (#540, @claycooper)

## v4.3.0

### Features / Enhancements

- Use Go 1.16
  - update dependencies, including kubernetes, docker, containerd and more
  - add `darwin/arm64` (Apple Silicon, M1) build target (#530)
  - use the new `//go:embed` feature to directly embed the jsonschema in the binary (#529)
- Add a status column to `k3d registry list` output (#496, @ebr)
- Allow non-prefixed (i.e. without `k3d-` prefix) user input when fetching resources (e.g. `k3d node get mycluster-server-0` would return successfully)

### Fixes

- Allow absolute paths for volumes on Windows (#510, @markrexwinkel)
- fix nil-pointer exception in case of non-existent IPAM network config
- Properly handle combinations of host/hostIP in kubeAPI settings reflected in the kubeconfig (#500, @fabricev)

### Misc

- docs: fix typo in stop command help text (#513, @searsaw)
- ci/ghaction: AUR (pre-)release now on Ubuntu 20.04 and latest archlinux image
- REMOVE incomplete and unused `containerd` runtime from codebase, as it was causing issues to build for windows and hasn't made any progress in quite some time now

## v4.2.0

### Features / Enhancements

- add processing step for cluster config, to configure it e.g. for hostnetwork mode (#477, @konradmalik)
- allow proxying UDP ports via the load balancer (#488, @k0da)

### Fixes

- fix usage of `DOCKER_HOST` env var for Kubeconfig server ref (trim port)
- fix error when trying to attach the same node (e.g. registry) to the same network twice (#486, @kuritka)
- fix Kube-API settings in configg file got overwritten (#490, @dtomasi)

### Misc

- add `k3d.version` label to created resources
- add Pull-Request template
- docs: add hint on minimal requirements for multi-server clusters (#481, @Filius-Patris)

## v4.1.1

### Fixes

- fix: `--k3s-server-arg` and `--k3s-agent-arg` didn't work (Viper StringArray incompatibility) (#482)

## v4.1.0

### Highlights

#### :scroll: Configuration Enhancements

- :snake: use [viper](https://github.com/spf13/viper) for configuration management
  - takes over the job of properly fetching and merging config options from
    - CLI arguments/flags
    - environment variables
    - config file
  - this also fixes some issues with using the config file (like cobra defaults overriding config file values)
- :heavy_check_mark: add JSON-Schema validation for the `Simple` config file schema
- :new: config version `k3d.io/v1alpha2` (some naming changes)
  - `exposeAPI` -> `kubeAPI`
  - `options.k3d.noRollback` -> `options.k3d.disableRollback`
  - `options.k3d.prepDisableHostIPInjection` -> `options.k3d.disableHostIPInjection`

#### :computer: Docker over SSH

- Support Docker over SSH (#324, @ekristen & @inercia)

### Features & Enhancements

- add root flag `--timestamps` to enable timestamped logs
- improved multi-server cluster support (#467)
  - log a warning, if one tries to create a cluster with only 2 nodes (no majority possible, no fault tolerance)
  - revamped cluster start procedure: init-node, sorted servers, agents, helpers
  - different log messages per role and start-place (that we wait for to consider a node to be ready)
  - module: `NodeStartOpts` now accept a `ReadyLogMessage` and `NodeState` now takes a `Started` timestamp string

### Fixes

- do not ignore `--no-hostip` flag and don't inject hostip if `--network=host` (#471, @konradmalik)
- fix: `--no-lb` ignored
- fix: print error cause when serverlb fails to start

### Misc

- tests/e2e: add config override test
- tests/e2e: add multi server start-stop cycle test
- tests/e2e: improved logs with stage and test details.
- builds&tests: use Docker 20.10 and BuildKit everywhere
- :memo: docs: add <https://github.com/AbsaOSS/k3d-action> (GitHub Action) as a related project (#476, @kuritka)

### Tested with

- E2E Tests ran with k3s versions
  - v1.17.17-k3s1 (see Known Issues below)
  - v1.18.15-k3s1 (see Known Issues below)
  - v1.19.7-k3s1
  - v1.20.2-k3s1

### Known Issues

- automatic multi-server cluster restarts tend to fail with k3s versions v1.17.x & v1.18.x and probably earlier versions (using dqlite)
- Using Viper brings us lots of nice features, but also one problem:
  - We had to switch StringArray flags to StringSlice flags, which
    - allow to use multiple flag values comma-separated in a single flag, but also
    - split flag values that contain a comma into separate parts (and we cannot handle issues that arise due to this)
      - so if you rely on commas in your flag values (e.g. for `--env X=a,b,c`), please consider filing an issue or supporting <https://github.com/spf13/viper/issues/246> and <https://github.com/spf13/viper/pull/398>
      - `--env X=a,b,c` would be treated the same as `--env X=a`, `--env b`, `--env c`

## v4.0.0

### Breaking Changes

#### Module

**If you're using k3d as a Go module, please have a look into the code to see all the changes!**

- We're open for chats via Slack or GitHub discussions

- Module is now on `github.com/rancher/k3d/v4` due to lots of breaking changes
- `pkg/cluster` is now `pkg/client`
- `ClusterCreate` and `NodeCreate` don't start the entities (containers) anymore
  - `ClusterRun` and `NodeRun` orchestrate the new Create and Start functionality
- `NodeDelete`/`ClusterDelete` now take an additional `NodeDeleteOpts`/`ClusterDeleteOpts` struct to toggle specific steps
- NodeSpec now features a list of networks (required for registries)
- New config flow: CLIConfig (SimpleConfig) -> ClusterConfig -> Cluster + Opts

#### CLI

- Some flags changed to also use `noun-action` syntax
  - e.g. `--switch-context --update-default-kubeconfig` -> `--kubeconfig-switch-context --kubeconfig-update-default`
  - this eases grouping and visibility

### Changes

#### Features

- **Registry Support**
  - k3d-managed registry like we had it in k3d v1.x
  - Option 1: default settings, paired with cluster creation
    - `k3d cluster create --registry-create` -> New registry for that cluster
    - `k3d cluster create --registry-use` -> Re-use existing registry
  - Option 2: customized, managed stand-alone
    - `k3d registry [create/start/stop/delete]`
    - Check the documentation, help text and tutorials for more details
  - Communicate managed registry using the LocalRegistryHostingV1 spec from [KEP-1755](https://github.com/kubernetes/enhancements/blob/0d69f7cea6fbe73a7d70fab569c6898f5ccb7be0/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry/README.md)
    - interesting especially for tools that reload images, like Tilt or Skaffold

- **Config File Support**
  - Put all your CLI-Arguments/Flags into a more readable config file and re-use it everywhere (keep it in your repo)
    - Note: this is not always a 1:1 matching in naming/syntax/semantics
  - `k3d cluster create --config myconfig.yaml`

    ```yaml
    apiVersion: k3d.io/v1alpha1
    kind: Simple
    name: mycluster
    servers: 3
    agents: 2
    ports:
      - port: 8080:80
        nodeFilters:
          - loadbalancer
    ```

  - Check out our test cases in [pkg/config/test_assets/](./pkg/config/test_assets/) for more config file examples
  - **Note**: The config file format (& feature) might still be a little rough around the edges and it's prone to change quickly until we hit a stable release of the config

- [WIP] Support for Lifecycle Hooks
  - Run any executable at specific stages during the cluster and node lifecycles
    - e.g. we modify the `registries.yaml` in the `preStart` stage of nodes
    - Guides will follow

- Print container creation time (#431, @inercia)
- add output formats for `cluster ls` and `node ls` (#439, @inercia)

#### Fixes

- import image: avoid nil pointer exception in specific cases
- cluster delete: properly handle node and network (#437)
- --port: fix bnil-pointer exception when exposing port on non-existent loadbalancer
- completion/zsh: source completion file

#### Misc

- Now building with Go 1.15
  - same for the k3d-tools code
- updated dependencies (including Docker v20.10)
- tests/e2e: add `E2E_INCLUDE` and rename `E2E_SKIP` to `E2E_EXCLUDE`
- tests/e2e: allow overriding the Helper Image Tag via `E2E_HELPER_IMAGE_TAG`
- docs: spell checking (#434, @jsoref)
- docs: add Chocolatey install option (#443, @erwinkersten)
