# [![k3d](docs/static/img/k3d_logo_black_blue.svg)](https://k3d.io/)

[![License](https://img.shields.io/github/license/k3d-io/k3d?style=flat-square)](./LICENSE.md)
![Downloads](https://img.shields.io/github/downloads/k3d-io/k3d/total.svg?style=flat-square)

[![Go Module](https://img.shields.io/badge/Go%20Module-github.com%2Fk3d--io%2Fk3d%2Fv5-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/k3d-io/k3d/v5)
[![Go version](https://img.shields.io/github/go-mod/go-version/k3d-io/k3d?logo=go&logoColor=white&style=flat-square)](./go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/k3d-io/k3d?style=flat-square)](https://goreportcard.com/report/github.com/k3d-io/k3d)

<!-- ALL-CONTRIBUTORS-BADGE:START - Do not remove or modify this section -->
[![All Contributors](https://img.shields.io/badge/all_contributors-28-orange.svg?style=flat-square)](#contributors-)
<!-- ALL-CONTRIBUTORS-BADGE:END -->
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v2.0%20adopted-ff69b4.svg)](code_of_conduct.md)

## [k3s in docker](https://k3d.io)

k3s is the lightweight Kubernetes distribution by Rancher: [k3s-io/k3s](https://github.com/k3s-io/k3s)

k3d creates containerized k3s clusters. This means, that you can spin up a multi-node k3s cluster on a single machine using docker.

[![asciicast](https://asciinema.org/a/436420.svg)](https://asciinema.org/a/436420)

**Note:** k3d is a **community-driven project** but it's not an official Rancher (SUSE) product.
**Sponsoring**: To spend any significant amount of time improving k3d, we rely on sponsorships:

  - [**GitHub Sponsors**: ![GitHub Sponsors](https://img.shields.io/github/sponsors/k3d-io?label=GitHub%20Sponsors&style=flat-square)](https://github.com/sponsors/k3d-io)
  - [**LiberaPay**: ![Liberapay patrons](https://img.shields.io/liberapay/patrons/k3d-io?label=Liberapay%20Patrons&style=flat-square)](https://liberapay.com/k3d-io)
  - [**IssueHunt**: ![IssueHunt](https://raw.githubusercontent.com/BoostIO/issuehunt-materials/refs/heads/master/v1/issuehunt-shield-v1.svg)](https://issuehunt.io/r/k3d-io/k3d)
## Learning

- Website with documentation: [k3d.io](https://k3d.io/)
- [Rancher Meetup - May 2020 - Simplifying Your Cloud-Native Development Workflow With K3s, K3c and K3d (YouTube)](https://www.youtube.com/watch?v=hMr3prm9gDM)
  - k3d demo repository: [k3d-io/k3d-demo](https://github.com/k3d-io/k3d-demo)

## Requirements

- [docker](https://docs.docker.com/install/)
  - Note: k3d v5.x.x requires at least Docker v20.10.5 (runc >= v1.0.0-rc93) to work properly (see [#807](https://github.com/k3d-io/k3d/issues/807))

## Releases

- May 2020: v1.7.x -> **v3.0.0** (rewrite)
- January 2021: v3.x.x -> **v4.0.0** (breaking changes)
- September 2021: v4.4.8 -> **v5.0.0** (breaking changes)

| Platform | Stage | Version | Release Date | Downloads so far |
|-----------------|--------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------|---|
| [**GitHub Releases**](https://github.com/k3d-io/k3d/releases) | stable | [![GitHub release (latest by date)](https://img.shields.io/github/v/release/k3d-io/k3d?label=%20&style=for-the-badge&logo=github)](https://github.com/k3d-io/k3d/releases/latest) | [![GitHub Release Date](https://img.shields.io/github/release-date/k3d-io/k3d?label=%20&style=for-the-badge)](https://github.com/k3d-io/k3d/releases/latest) | ![GitHub Release Downloads](https://img.shields.io/github/downloads/k3d-io/k3d/latest/total?label=%20&style=for-the-badge) |
| [**GitHub Releases**](https://github.com/k3d-io/k3d/releases) | latest | [![GitHub release (latest by date including pre-releases)](https://img.shields.io/github/v/release/k3d-io/k3d?include_prereleases&label=%20&style=for-the-badge&logo=github)](https://github.com/k3d-io/k3d/releases) | [![GitHub Release Date](https://img.shields.io/github/release-date/k3d-io/k3d?include_prereleases&label=%20&style=for-the-badge)](https://github.com/k3d-io/k3d/releases) | ![GitHub Release Downloads](https://img.shields.io/github/downloads/k3d-io/k3d/latest/total?include_prereleases&label=%20&style=for-the-badge) |
| linux/amd64 | stable | [![GitHub release (latest by date)](https://img.shields.io/github/v/release/k3d-io/k3d?label=%20&style=for-the-badge&logo=github)](https://github.com/k3d-io/k3d/releases/latest) | [![GitHub Release Date](https://img.shields.io/github/release-date/k3d-io/k3d?label=%20&style=for-the-badge)](https://github.com/k3d-io/k3d/releases/latest) | ![GitHub Release Downloads](https://img.shields.io/github/downloads/k3d-io/k3d/latest/total?label=%20&style=for-the-badge) |
| linux/arm64 | stable | [![GitHub release (latest by date)](https://img.shields.io/github/v/release/k3d-io/k3d?label=%20&style=for-the-badge&logo=github)](https://github.com/k3d-io/k3d/releases/latest) | [![GitHub Release Date](https://img.shields.io/github/release-date/k3d-io/k3d?label=%20&style=for-the-badge)](https://github.com/k3d-io/k3d/releases/latest) | ![GitHub Release Downloads](https://img.shields.io/github/downloads/k3d-io/k3d/latest/total?label=%20&style=for-the-badge) |
| linux/riscv64 | stable | [![GitHub release (latest by date)](https://img.shields.io/github/v/release/k3d-io/k3d?label=%20&style=for-the-badge&logo=github)](https://github.com/k3d-io/k3d/releases/latest) | [![GitHub Release Date](https://img.shields.io/github/release-date/k3d-io/k3d?label=%20&style=for-the-badge)](https://github.com/k3d-io/k3d/releases/latest) | ![GitHub Release Downloads](https://img.shields.io/github/downloads/k3d-io/k3d/latest/total?label=%20&style=for-the-badge) |