#!/usr/bin/env bash

APP_NAME="k3d"
REPO_URL="https://github.com/k3d-io/k3d"

: ${USE_SUDO:="true"}
: ${K3D_INSTALL_DIR:="/usr/local/bin"}

# initArch discovers the architecture for this system.
initArch() {
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
}

# initOS discovers the operating system for this system.
initOS() {
  OS=$(uname|tr '[:upper:]' '[:lower:]')

  case "$OS" in
    # Minimalist GNU for Windows
    mingw*) OS='windows';;
  esac
}

# runs the given command as root (detects if we are root already)
runAsRoot() {
  local CMD="$*"

  if [ $EUID -ne 0 -a $USE_SUDO = "true" ]; then
    CMD="sudo $CMD"
  fi

  $CMD
}

# scurl invokes `curl` with secure defaults
scurl() {
  # - `--proto =https` requires that all URLs use HTTPS. Attempts to call http://
  #   URLs will fail.
  # - `--tlsv1.2` ensures that at least TLS v1.2 is used, disabling less secure
  #   prior TLS versions.
  # - `--fail` ensures that the command fails if HTTP response is not 2xx.
  # - `--show-error` causes curl to output error messages when it fails (when
  #   also invoked with -s|--silent).
  curl --proto "=https" --tlsv1.2 --fail --show-error "$@"
}

# verifySupported checks that the os/arch combination is supported for
# binary builds.
verifySupported() {
  local supported="darwin-386\ndarwin-amd64\ndarwin-arm64\nlinux-386\nlinux-amd64\nlinux-arm\nlinux-arm64\nwindows-386\nwindows-amd64"
  if ! echo "${supported}" | grep -q "${OS}-${ARCH}"; then
    echo "No prebuilt binary for ${OS}-${ARCH}."
    echo "To build from source, go to $REPO_URL"
    exit 1
  fi

  if ! type "curl" > /dev/null && ! type "wget" > /dev/null; then
    echo "Either curl or wget is required"
    exit 1
  fi
}

# checkK3dInstalledVersion checks which version of k3d is installed and
# if it needs to be changed.
checkK3dInstalledVersion() {
  if [[ -f "${K3D_INSTALL_DIR}/${APP_NAME}" ]]; then
    local version=$(k3d version | grep 'k3d version' | cut -d " " -f3)
    if [[ "$version" == "$TAG" ]]; then
      echo "k3d ${version} is already ${DESIRED_VERSION:-latest}"
      return 0
    else
      echo "k3d ${TAG} is available. Changing from version ${version}."
      return 1
    fi
  else
    return 1
  fi
}

# checkTagProvided checks whether TAG has provided as an environment variable so we can skip checkLatestVersion.
checkTagProvided() {
  [[ ! -z "$TAG" ]]
}

# checkLatestVersion grabs the latest version string from the releases
checkLatestVersion() {
  local latest_release_url="$REPO_URL/releases/latest"
  if type "curl" > /dev/null; then
    TAG=$(scurl -Ls -o /dev/null -w %{url_effective} $latest_release_url | grep -oE "[^/]+$" )
  elif type "wget" > /dev/null; then
    TAG=$(wget $latest_release_url --server-response -O /dev/null 2>&1 | awk '/^\s*Location: /{DEST=$2} END{ print DEST}' | grep -oE "[^/]+$")
  fi
}

# downloadFile downloads the latest binary package and also the checksum
# for that binary.
downloadFile() {
  K3D_DIST="k3d-$OS-$ARCH"
  DOWNLOAD_URL="$REPO_URL/releases/download/$TAG/$K3D_DIST"
  K3D_TMP_ROOT="$(mktemp -dt k3d-binary-XXXXXX)"
  K3D_TMP_FILE="$K3D_TMP_ROOT/$K3D_DIST"
  if type "curl" > /dev/null; then
    scurl -sL "$DOWNLOAD_URL" -o "$K3D_TMP_FILE"
  elif type "wget" > /dev/null; then
    wget -q -O "$K3D_TMP_FILE" "$DOWNLOAD_URL"
  fi
}

# installFile verifies the SHA256 for the file, then unpacks and
# installs it.
installFile() {
  echo "Preparing to install $APP_NAME into ${K3D_INSTALL_DIR}"
  runAsRoot chmod +x "$K3D_TMP_FILE"
  runAsRoot cp "$K3D_TMP_FILE" "$K3D_INSTALL_DIR/$APP_NAME"
  echo "$APP_NAME installed into $K3D_INSTALL_DIR/$APP_NAME"
}

# fail_trap is executed if an error occurs.
fail_trap() {
  result=$?
  if [ "$result" != "0" ]; then
    if [[ -n "$INPUT_ARGUMENTS" ]]; then
      echo "Failed to install $APP_NAME with the arguments provided: $INPUT_ARGUMENTS"
      help
    else
      echo "Failed to install $APP_NAME"
    fi
    echo -e "\tFor support, go to $REPO_URL."
  fi
  cleanup
  exit $result
}

# testVersion tests the installed client to make sure it is working.
testVersion() {
  if ! command -v $APP_NAME &> /dev/null; then
    echo "$APP_NAME not found. Is $K3D_INSTALL_DIR on your "'$PATH?'
    exit 1
  fi
  echo "Run '$APP_NAME --help' to see what you can do with it."
}

# help provides possible cli installation arguments
help () {
  echo "Accepted cli arguments are:"
  echo -e "\t[--help|-h ] ->> prints this help"
  echo -e "\t[--no-sudo]  ->> install without sudo"
}

# cleanup temporary files
cleanup() {
  if [[ -d "${K3D_TMP_ROOT:-}" ]]; then
    rm -rf "$K3D_TMP_ROOT"
  fi
}

# Execution

#Stop execution on any error
trap "fail_trap" EXIT
set -e

# Parsing input arguments (if any)
export INPUT_ARGUMENTS="${@}"
set -u
while [[ $# -gt 0 ]]; do
  case $1 in
    '--no-sudo')
       USE_SUDO="false"
       ;;
    '--help'|-h)
       help
       exit 0
       ;;
    *) exit 1
       ;;
  esac
  shift
done
set +u

initArch
initOS
verifySupported
checkTagProvided || checkLatestVersion
if ! checkK3dInstalledVersion; then
  downloadFile
  installFile
fi
testVersion
cleanup
