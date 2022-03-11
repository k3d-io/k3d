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
  inherits = ["docker-metadata-action"]
  platforms = ["linux/amd64", "linux/arm64", "linux/arm/v7"]
}

target "binary" {
  inherits = ["default-release-options"]
  dockerfile = "Dockerfile"
  context = "."
  target = "binary-only"
}

target "dind" {
  inherits = ["default-release-options"]
  dockerfile = "Dockerfile"
  context = "."
  target = "dind"
}

target "proxy" {
  context = "proxy/"
}

target "tools" {
  context = "tools/"
}
