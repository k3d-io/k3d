#!/bin/sh

# initArch discovers the architecture for this system.
initArch() {
  if [ -z $ARCH  ]; then
    ARCH=$(uname -m)
    case $ARCH in
      armv5*) ARCH="armv5";;
      armv6*) ARCH="armv6";;
      armv7*) ARCH="arm";;
      aarch64) ARCH="arm64";;
      x86) ARCH="386";;
      x86_64) ARCH="amd64";;
      i686) ARCH="386";;
      i386) ARCH="386";;
    esac
  fi
}

# initOS discovers the operating system for this system.
initOS() {
  if [ -z $OS ]; then
    OS=$(uname|tr '[:upper:]' '[:lower:]')

    case "$OS" in
      # Minimalist GNU for Windows
      mingw*) OS='windows';;
    esac
  fi
}

install_kubectl() {
  echo "Installing kubectl for $OS/$ARCH..."
  curl -sSfL "https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/${OS}/${ARCH}/kubectl" -o ./kubectl
  chmod +x ./kubectl
	mv ./kubectl /usr/local/bin/kubectl
}

install_yq() {
  echo "Installing yq for $OS/$ARCH..."
  curl -sSfL https://github.com/mikefarah/yq/releases/download/v4.9.6/yq_${OS}_${ARCH} -o ./yq
  chmod +x ./yq
  mv ./yq /usr/local/bin/yq
}

install_golangci_lint() {
  echo "Installing golangci-lint..."
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.49.0
}

install_gox() {
  echo "Installing gox for $OS/$ARCH..."
  GOX_REPO=iwilltry42/gox
  GOX_VERSION=0.1.0
  curl -sSfL https://github.com/${GOX_REPO}/releases/download/v${GOX_VERSION}/gox_${GOX_VERSION}_${OS}_${ARCH}.tar.gz | tar -xz -C /tmp
  chmod +x /tmp/gox
  mv /tmp/gox /usr/local/bin/gox
}

install_confd() {
  echo "Installing confd for $OS/$ARCH..."
  CONFD_REPO=iwilltry42/confd
  CONFD_VERSION=0.17.0-rc.0
  curl -sSfL "https://github.com/${CONFD_REPO}/releases/download/v${CONFD_VERSION}/confd-${CONFD_VERSION}-${OS}-${ARCH}" -o ./confd
  chmod +x ./confd
  mv ./confd /usr/local/bin/confd
}


#
# MAIN
#

initOS
initArch

for pkg in "$@"; do
  case "$pkg" in
    kubectl) install_kubectl;;
    yq) install_yq;;
    golangci-lint) install_golangci_lint;;
    confd) install_confd;;
    gox) install_gox;;
    *) printf "ERROR: Unknown Package '%s'" $pkg;;
  esac
done
