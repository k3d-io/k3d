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
install_confd
