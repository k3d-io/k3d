#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

: "${E2E_SKIP:=""}"
: "${E2E_EXTRA:=""}"

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

#########################################################################################

[ -n "$EXE" ] || abort "no EXE provided"

info "Preparing filesystem and environment..."

mkdir -p $HOME/.kube

section "BASIC TESTS"

for i in $CURR_DIR/test_*.sh ; do
  base=$(basename "$i" .sh)
  skip=false
  for skiptest in "${E2E_SKIP[@]}"; do
    [[ "$skiptest" =~ (^| )${base}($| ) ]] && skip=true
  done
  if [ "$skip" = true ]; then
    highlight "***** Skipping $base *****"
  else
    highlight "***** Running $base *****"
    "$i" || abort "test $base failed"
  fi
done

if [[ -n "$E2E_EXTRA" ]]; then
  section "EXTRA TESTS"
  for i in $CURR_DIR/extra_test_*.sh ; do
    base=$(basename "$i" .sh)
    if [[ $E2E_SKIP =~ (^| )$base($| ) ]]; then
      highlight "***** Skipping $base *****"
    else
      highlight "***** Running $base *****"
      "$i" || abort "test $base failed"
    fi
  done
else
  info "NOT running EXTRA tests, please set E2E_EXTRA=1 if you wish to do so"
fi

exit 0

