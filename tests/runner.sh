#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

: "${E2E_INCLUDE:=""}"
: "${E2E_EXCLUDE:=""}"
: "${E2E_EXTRA:=""}"

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

#########################################################################################

[ -n "$EXE" ] || abort "no EXE provided"

info "Preparing filesystem and environment..."

mkdir -p $HOME/.kube

echo "Start time inside runner: $(date)"

section "BASIC TESTS"

for i in $CURR_DIR/test_*.sh ; do
  base=$(basename "$i" .sh)
  skip=false
  included=false
  excluded=false

  # prepare to skip test, if it's in the exclusion list
  for excludetest in "${E2E_EXCLUDE[@]}"; do
    [[ "$excludetest" =~ (^| )${base}($| ) ]] && excluded=true
  done

  # (re-)add test to list, if it's on inclusion list
  for includetest in "${E2E_INCLUDE[@]}"; do
    [[ "$includetest" =~ (^| )${base}($| ) ]] && included=true
  done

  if  [[ -z "${E2E_INCLUDE}" ]]; then # no explicit include list given
    if $excluded; then # test is on explicit exclude list
      skip=true
    fi
  else
    if $included && $excluded; then # test is in both lists, so we include it
      warn "Test ${base} is in both, exclude and include list. Include list takes precedence."
      skip=false
    fi
    if ! $included; then # test is not in include list -> skip
      skip=true
    fi
  fi

  # skip or run test
  if [ "$skip" = true ]; then
    highlight "***** Skipping $base *****"
  else
    highlight "***** Running $base *****"
    "$i" || abort "test $base failed"
  fi
done

# Additional (extra) tests
if [[ -n "$E2E_EXTRA" ]]; then
  section "EXTRA TESTS"
  for i in $CURR_DIR/extra_test_*.sh ; do
    base=$(basename "$i" .sh)
    if [[ $E2E_EXCLUDE =~ (^| )$base($| ) ]]; then
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

