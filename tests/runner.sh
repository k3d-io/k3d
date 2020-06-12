#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

: "${E2E_SKIP:=""}"

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

#########################################################################################

[ -n "$EXE" ] || abort "no EXE provided"

info "Preparing filesystem and environment..."

mkdir -p $HOME/.kube

for i in $CURR_DIR/test_*.sh ; do
  base=$(basename $i .sh)
  if [[ $E2E_SKIP =~ (^| )$base($| ) ]]; then
    highlight "***** Skipping $base *****"
  else
    highlight "***** Running $base *****"
    $i || abort "test $base failed"
  fi
done

exit 0

