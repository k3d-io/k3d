# [![k3d](docs/static/img/k3d_logo_black_blue.svg)](https://k3d.io/)

[![License](https://img.shields.io/github/license/k3d-io/k3d?style=flat-square)](./LICENSE.md)
![Downloads](https://img.shields.io/github/downloads/k3d-io/k3d/total.svg?style=flat-square)

[![Go Module](https://img.shields.io/badge/Go%20Module-github.com%k3d-io%2Fk3d%2Fv5-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/k3d-io/k3d/v5)
[![Go version](https://img.shields.io/github/go-mod/go-version/k3d-io/k3d?logo=go&logoColor=white&style=flat-square)](./go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/k3d-io/k3d?style=flat-square)](https://goreportcard.com/report/github.com/k3d-io/k3d)

<!-- ALL-CONTRIBUTORS-BADGE:START - Do not remove or modify this section -->
[![All Contributors](https://img.shields.io/badge/all_contributors-27-orange.svg?style=flat-square)](#contributors-)
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
  - **IssueHunt**: <https://issuehunt.io/r/k3d-io/k3d>

## Learning

- Website with documentation: [k3d.io](https://k3d.io/)
- [Rancher Meetup - May 2020 - Simplifying Your Cloud-Native Development Workflow With K3s, K3c and K3d (YouTube)](https://www.youtube.com/watch?v=hMr3prm9gDM)
  - k3d demo repository: [iwilltry42/k3d-demo](https://github.com/iwilltry42/k3d-demo)

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
| [**GitHub Releases**](https://github.com/k3d-io/k3d/releases) | latest | [![GitHub release (latest by date including pre-releases)](https://img.shields.io/github/v/release/k3d-io/k3d?include_prereleases&label=%20&style=for-the-badge&logo=github)](https://github.com/k3d-io/k3d/releases) | [![GitHub (Pre-)Release Date](https://img.shields.io/github/release-date-pre/k3d-io/k3d?label=%20&style=for-the-badge)](https://github.com/k3d-io/k3d/releases) | ![GitHub Release Downloads (incl. Pre-Releases)](https://img.shields.io/github/downloads-pre/k3d-io/k3d/latest/total?label=%20&style=for-the-badge) |
| [**Homebrew**](https://formulae.brew.sh/formula/k3d) | stable | [![homebrew](https://img.shields.io/homebrew/v/k3d?label=%20&style=for-the-badge)](https://formulae.brew.sh/formula/k3d) | - | - |
| [**Chocolatey**](https://chocolatey.org/packages/k3d/)| stable | [![chocolatey](https://img.shields.io/chocolatey/v/k3d?label=%20&style=for-the-badge)](https://chocolatey.org/packages/k3d/) | - | - |
| [**Scoop**](https://github.com/ScoopInstaller/Main/blob/master/bucket/k3d.json/)| stable | [![scoop](https://img.shields.io/scoop/v/k3d?label=%20&style=for-the-badge)](https://github.com/ScoopInstaller/Main/blob/master/bucket/k3d.json/) | - | - |

## Get

You have several options there:

- use the install script to grab the latest release:
  - wget: `wget -q -O - https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash`
  - curl: `curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash`
- use the install script to grab a specific release (via `TAG` environment variable):
  - wget: `wget -q -O - https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | TAG=v5.0.0 bash`
  - curl: `curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | TAG=v5.0.0 bash`

- use [Homebrew](https://brew.sh): `brew install k3d` (Homebrew is available for MacOS and Linux)
  - Formula can be found in [homebrew/homebrew-core](https://github.com/Homebrew/homebrew-core/blob/master/Formula/k3d.rb) and is mirrored to [homebrew/linuxbrew-core](https://github.com/Homebrew/linuxbrew-core/blob/master/Formula/k3d.rb)
- install via [MacPorts](https://www.macports.org): `sudo port selfupdate && sudo port install k3d` (MacPorts is available for MacOS)
- install via [AUR](https://aur.archlinux.org/) package [rancher-k3d-bin](https://aur.archlinux.org/packages/rancher-k3d-bin/): `yay -S rancher-k3d-bin`
- grab a release from the [release tab](https://github.com/k3d-io/k3d/releases) and install it yourself.
- install via go: `go install github.com/k3d-io/k3d/v5@latest` (**Note**: this will give you unreleased/bleeding-edge changes)
- use [Chocolatey](https://chocolatey.org/): `choco install k3d` (Chocolatey package manager is available for Windows)
  - package source can be found in [erwinkersten/chocolatey-packages](https://github.com/erwinkersten/chocolatey-packages/tree/master/automatic/k3d)
- use [Scoop](https://scoop.sh/): `scoop install k3d` (Scoop package manager is available for Windows)
  - package source can be found in [ScoopInstaller/Main](https://github.com/ScoopInstaller/Main/blob/master/bucket/k3d.json)

or...

## Build

1. Clone this repo, e.g. via `git clone git@github.com:k3d-io/k3d.git` or `go get github.com/k3d-io/k3d/v5@main`
2. Inside the repo run
   - 'make install-tools' to make sure required go packages are installed
3. Inside the repo run one of the following commands
   - `make build` to build for your current system
   - `go install` to install it to your `GOPATH` (**Note**: this will give you unreleased/bleeding-edge changes)
   - `make build-cross` to build for all systems

## Usage

Check out what you can do via `k3d help` or check the docs @ [k3d.io](https://k3d.io)

Example Workflow: Create a new cluster and use it with `kubectl`

1. `k3d cluster create CLUSTER_NAME` to create a new single-node cluster (= 1 container running k3s + 1 loadbalancer container)
2. [Optional, included in cluster create] `k3d kubeconfig merge CLUSTER_NAME --kubeconfig-switch-context` to update your default kubeconfig and switch the current-context to the new one
3. execute some commands like `kubectl get pods --all-namespaces`
4. `k3d cluster delete CLUSTER_NAME` to delete the default cluster

## Connect

1. Join the Rancher community on slack via [slack.rancher.io](https://slack.rancher.io/)
2. Go to [rancher-users.slack.com](https://rancher-users.slack.com) and join our channel #k3d
3. Start chatting

## History

This repository is based on [@zeerorg](https://github.com/zeerorg/)'s [zeerorg/k3s-in-docker](https://github.com/zeerorg/k3s-in-docker), reimplemented in Go by [@iwilltry42](https://github.com/iwilltry42/) in [iwilltry42/k3d](https://github.com/iwilltry42/k3d), which got adopted by Rancher in [rancher/k3d](https://github.com/rancher/k3d) and was now moved into its own GitHub organization at [k3d-io/k3d](https://github.com/k3d-io/k3d).

## Related Projects

- [k3x](https://github.com/inercia/k3x): GUI (Linux) to k3d
- [vscode-k3d](https://github.com/inercia/vscode-k3d): vscode plugin for k3d
- [AbsaOSS/k3d-action](https://github.com/AbsaOSS/k3d-action): fully customizable GitHub Action to run lightweight Kubernetes clusters.
- [AutoK3s](https://github.com/cnrancher/autok3s): a lightweight tool to help run K3s everywhere including k3d provider.
- [nolar/setup-k3d-k3s](https://github.com/nolar/setup-k3d-k3s): setup K3d/K3s for GitHub Actions.

## Contributing

k3d is a community-driven project and so we welcome contributions of any form, be it code, logic, documentation, examples, requests, bug reports, ideas or anything else that pushes this project forward.

Please read our [**Contributing Guidelines**](./CONTRIBUTING.md) and the related [**Code of Conduct**](./CODE_OF_CONDUCT.md).

You can find an overview of the k3d project (e.g. explanations and a repository guide) in the documentation: [k3d.io/stable/design/project/](https://k3d.io/stable/design/project/)

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v2.0%20adopted-ff69b4.svg)](code_of_conduct.md)

## Contributors ✨

Thanks goes to these wonderful people ([emoji key](https://allcontributors.org/docs/en/emoji-key)):

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore-start -->
<!-- markdownlint-disable -->
<table>
  <tbody>
    <tr>
      <td align="center" valign="top" width="14.28%"><a href="https://twitter.com/iwilltry42"><img src="https://avatars3.githubusercontent.com/u/25345277?v=4?s=100" width="100px;" alt="Thorsten Klein"/><br /><sub><b>Thorsten Klein</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=iwilltry42" title="Code">💻</a> <a href="https://github.com/k3d-io/k3d/commits?author=iwilltry42" title="Documentation">📖</a> <a href="#ideas-iwilltry42" title="Ideas, Planning, & Feedback">🤔</a> <a href="#maintenance-iwilltry42" title="Maintenance">🚧</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://blog.zeerorg.site/"><img src="https://avatars0.githubusercontent.com/u/13547997?v=4?s=100" width="100px;" alt="Rishabh Gupta"/><br /><sub><b>Rishabh Gupta</b></sub></a><br /><a href="#ideas-zeerorg" title="Ideas, Planning, & Feedback">🤔</a> <a href="https://github.com/k3d-io/k3d/commits?author=zeerorg" title="Code">💻</a></td>
      <td align="center" valign="top" width="14.28%"><a href="http://www.zenika.com"><img src="https://avatars3.githubusercontent.com/u/25585516?v=4?s=100" width="100px;" alt="Louis Tournayre"/><br /><sub><b>Louis Tournayre</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=louiznk" title="Documentation">📖</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/lionelnicolas"><img src="https://avatars3.githubusercontent.com/u/6538664?v=4?s=100" width="100px;" alt="Lionel Nicolas"/><br /><sub><b>Lionel Nicolas</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=lionelnicolas" title="Code">💻</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/toonsevrin.keys"><img src="https://avatars1.githubusercontent.com/u/5507199?v=4?s=100" width="100px;" alt="Toon Sevrin"/><br /><sub><b>Toon Sevrin</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=toonsevrin" title="Code">💻</a></td>
      <td align="center" valign="top" width="14.28%"><a href="http://debian-solutions.de"><img src="https://avatars3.githubusercontent.com/u/1111056?v=4?s=100" width="100px;" alt="Dennis Hoppe"/><br /><sub><b>Dennis Hoppe</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=dhoppe" title="Documentation">📖</a> <a href="#example-dhoppe" title="Examples">💡</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://dellinger.dev"><img src="https://avatars0.githubusercontent.com/u/3109892?v=4?s=100" width="100px;" alt="Jonas Dellinger"/><br /><sub><b>Jonas Dellinger</b></sub></a><br /><a href="#infra-JohnnyCrazy" title="Infrastructure (Hosting, Build-Tools, etc)">🚇</a></td>
    </tr>
    <tr>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/markrexwinkel"><img src="https://avatars2.githubusercontent.com/u/10704814?v=4?s=100" width="100px;" alt="markrexwinkel"/><br /><sub><b>markrexwinkel</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=markrexwinkel" title="Documentation">📖</a></td>
      <td align="center" valign="top" width="14.28%"><a href="http://inerciatech.com/"><img src="https://avatars2.githubusercontent.com/u/1841612?v=4?s=100" width="100px;" alt="Alvaro"/><br /><sub><b>Alvaro</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=inercia" title="Code">💻</a> <a href="#ideas-inercia" title="Ideas, Planning, & Feedback">🤔</a> <a href="#plugin-inercia" title="Plugin/utility libraries">🔌</a></td>
      <td align="center" valign="top" width="14.28%"><a href="http://wsl.dev"><img src="https://avatars2.githubusercontent.com/u/905874?v=4?s=100" width="100px;" alt="Nuno do Carmo"/><br /><sub><b>Nuno do Carmo</b></sub></a><br /><a href="#content-nunix" title="Content">🖋</a> <a href="#tutorial-nunix" title="Tutorials">✅</a> <a href="#question-nunix" title="Answering Questions">💬</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/erwinkersten"><img src="https://avatars0.githubusercontent.com/u/4391121?v=4?s=100" width="100px;" alt="Erwin Kersten"/><br /><sub><b>Erwin Kersten</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=erwinkersten" title="Documentation">📖</a></td>
      <td align="center" valign="top" width="14.28%"><a href="http://www.alexsears.com"><img src="https://avatars.githubusercontent.com/u/3712883?v=4?s=100" width="100px;" alt="Alex Sears"/><br /><sub><b>Alex Sears</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=searsaw" title="Documentation">📖</a></td>
      <td align="center" valign="top" width="14.28%"><a href="http://shanduur.github.io"><img src="https://avatars.githubusercontent.com/u/32583062?v=4?s=100" width="100px;" alt="Mateusz Urbanek"/><br /><sub><b>Mateusz Urbanek</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=Shanduur" title="Code">💻</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/benjaminjb"><img src="https://avatars.githubusercontent.com/u/4651855?v=4?s=100" width="100px;" alt="Benjamin Blattberg"/><br /><sub><b>Benjamin Blattberg</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=benjaminjb" title="Code">💻</a></td>
    </tr>
    <tr>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/sbaier1"><img src="https://avatars.githubusercontent.com/u/20339314?v=4?s=100" width="100px;" alt="Simon Baier"/><br /><sub><b>Simon Baier</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=sbaier1" title="Code">💻</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://ambrose.website"><img src="https://avatars.githubusercontent.com/u/1705906?v=4?s=100" width="100px;" alt="Ambrose Chua"/><br /><sub><b>Ambrose Chua</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=serverwentdown" title="Code">💻</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://www.zenior.no/"><img src="https://avatars.githubusercontent.com/u/1142578?v=4?s=100" width="100px;" alt="Erik Godding Boye"/><br /><sub><b>Erik Godding Boye</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=erikgb" title="Code">💻</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/ethinx"><img src="https://avatars.githubusercontent.com/u/965612?v=4?s=100" width="100px;" alt="York Wong"/><br /><sub><b>York Wong</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=ethinx" title="Code">💻</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/gonzalesraul"><img src="https://avatars.githubusercontent.com/u/7568984?v=4?s=100" width="100px;" alt="Raul Gonzales"/><br /><sub><b>Raul Gonzales</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=gonzalesraul" title="Code">💻</a> <a href="https://github.com/k3d-io/k3d/commits?author=gonzalesraul" title="Documentation">📖</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://hoon.dev"><img src="https://avatars.githubusercontent.com/u/6194958?v=4?s=100" width="100px;" alt="Sunghoon Kang"/><br /><sub><b>Sunghoon Kang</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=devholic" title="Code">💻</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://blogs.kameshs.dev"><img src="https://avatars.githubusercontent.com/u/947745?v=4?s=100" width="100px;" alt="Kamesh Sampath"/><br /><sub><b>Kamesh Sampath</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=kameshsampath" title="Code">💻</a></td>
    </tr>
    <tr>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/arikmaor"><img src="https://avatars.githubusercontent.com/u/1926243?v=4?s=100" width="100px;" alt="Arik Maor"/><br /><sub><b>Arik Maor</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=arikmaor" title="Code">💻</a> <a href="#tutorial-arikmaor" title="Tutorials">✅</a> <a href="#example-arikmaor" title="Examples">💡</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://radiusmethod.com"><img src="https://avatars.githubusercontent.com/u/1850811?v=4?s=100" width="100px;" alt="Danny Gershman"/><br /><sub><b>Danny Gershman</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=dgershman" title="Code">💻</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/stopanko"><img src="https://avatars.githubusercontent.com/u/3759349?v=4?s=100" width="100px;" alt="stopanko"/><br /><sub><b>stopanko</b></sub></a><br /><a href="#financial-stopanko" title="Financial">💵</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/dbreyfogle"><img src="https://avatars.githubusercontent.com/u/27653146?v=4?s=100" width="100px;" alt="Danny Breyfogle"/><br /><sub><b>Danny Breyfogle</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=dbreyfogle" title="Documentation">📖</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://tech.aabouzaid.com/"><img src="https://avatars.githubusercontent.com/u/6760103?v=4?s=100" width="100px;" alt="Ahmed AbouZaid"/><br /><sub><b>Ahmed AbouZaid</b></sub></a><br /><a href="#ideas-aabouzaid" title="Ideas, Planning, & Feedback">🤔</a> <a href="https://github.com/k3d-io/k3d/commits?author=aabouzaid" title="Code">💻</a> <a href="https://github.com/k3d-io/k3d/commits?author=aabouzaid" title="Documentation">📖</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://roudier.io"><img src="https://avatars.githubusercontent.com/u/1056127?v=4?s=100" width="100px;" alt="Pierre Roudier"/><br /><sub><b>Pierre Roudier</b></sub></a><br /><a href="https://github.com/k3d-io/k3d/commits?author=proudier" title="Code">💻</a></td>
    </tr>
  </tbody>
</table>

<!-- markdownlint-restore -->
<!-- prettier-ignore-end -->

<!-- ALL-CONTRIBUTORS-LIST:END -->

This project follows the [all-contributors](https://github.com/all-contributors/all-contributors) specification. Contributions of any kind welcome!

## Sponsors

Thanks to all our amazing sponsors! 🙏

<ul>
  <!-- sponsors --><!-- sponsors -->
</ul>
