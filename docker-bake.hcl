// release group
group "release" {
  targets = ["binary", "dind", "proxy", "tools"]
}

// filled by GitHub Actions
target "docker-metadata-k3d" {}
target "docker-metadata-k3d-dind" {}
target "docker-metadata-k3d-proxy" {}
target "docker-metadata-k3d-tools" {}

// default options for creating a release
target "default-release-options" {
  platforms = ["linux/amd64", "linux/arm64", "linux/arm/v7"]
}

target "binary" {
  inherits = ["default-release-options", "docker-metadata-k3d"]
  dockerfile = "Dockerfile"
  context = "."
  target = "binary-only"
}

target "dind" {
  inherits = ["docker-metadata-k3d-dind"] // dind does not inherit defaults, as dind base image is not available for armv7
  platforms = ["linux/amd64", "linux/arm64"]
  dockerfile = "Dockerfile"
  context = "."
  target = "dind"
}

target "proxy" {
  inherits = ["default-release-options", "docker-metadata-k3d-proxy"]
  context = "proxy/"
}

target "tools" {
  inherits = ["default-release-options", "docker-metadata-k3d-tools"]
  context = "tools/"
}
