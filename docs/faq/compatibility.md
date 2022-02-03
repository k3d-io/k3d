# Compatibility

With each release, we test if k3d works with specific versions of Docker and K3s, to ensure, that at least the most recent versions of Docker and the active releases (i.e. non-EOL release channels, similar to Kubernetes) work properly with it.
The tests happen automatically in GitHub Actions.
Some versions of Docker and K3s are expected to fail with specific versions of k3d due to e.g. incompatible dependencies or missing features.
We test a full cluster lifecycle with different [K3s channels](https://update.k3s.io/v1-release/channels), meaning that the following list refers to the current latest version released under the given channel.

## Releases

### v5.3.0 - 03.02.2022

#### Docker

* 20.10.5
* 20.10.12

**Expected to Fail** with the following versions:

* <= 20.10.4 (due to runc, see <https://github.com/rancher/k3d/issues/807>)

#### K3s

* Channel v1.23
* Channel v1.22

**Expected to Fail** with the following versions:

* <= v1.18 (due to not included, but expected CoreDNS in K3s)
