// release group
group "release" {
  targets = ["binary", "dind", "proxy", "tools"]
}

// filled by GitHub Actions
target "docker-metadata-k3d" {}
target "docker-metadata-k3d-dind" {}
target "docker-metadata-k3d-proxy" {}
target "docker-metadata-k3d-tools" {}

// version embedded into the k3d and k3d-tools binaries
// -> filled by GitHub Actions, as the build context has no .git directory to derive it from
// -> left empty, the build falls back to `git describe`
variable "GIT_TAG" {
  default = ""
}

// default options for creating a release
target "default-release-options" {
  platforms = ["linux/amd64", "linux/arm64", "linux/arm/v7"]
}

target "binary" {
  inherits = ["default-release-options", "docker-metadata-k3d"]
  dockerfile = "Dockerfile"
  context = "."
  target = "binary-only"
  args = {
    GIT_TAG_OVERRIDE = GIT_TAG
  }
}

target "dind" {
  inherits = ["docker-metadata-k3d-dind"] // dind does not inherit defaults, as dind base image is not available for armv7
  platforms = ["linux/amd64", "linux/arm64"]
  dockerfile = "Dockerfile"
  context = "."
  target = "dind"
  args = {
    GIT_TAG_OVERRIDE = GIT_TAG
  }
}

target "proxy" {
  inherits = ["default-release-options", "docker-metadata-k3d-proxy"]
  context = "proxy/"
}

target "tools" {
  inherits = ["default-release-options", "docker-metadata-k3d-tools"]
  context = "tools/"
  args = {
    GIT_TAG = GIT_TAG
  }
}
