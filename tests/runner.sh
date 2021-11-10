#!/bin/bash

shopt -s nullglob # nullglob expands non-matching globs to zero arguments, rather than to themselves -> empty array when used to get array of filepaths

CURR_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
[ -d "$CURR_DIR" ] || {
  echo "FATAL: no current dir (maybe running in zsh?)"
  exit 1
}

: "${E2E_INCLUDE:=""}"
: "${E2E_EXCLUDE:=""}"
: "${E2E_EXTRA:=""}"
: "${E2E_PARALLEL:="4"}"

export CURRENT_STAGE="Runner"

# shellcheck disable=SC1091
source "$CURR_DIR/common.sh"

#########################################################################################

[ -n "$EXE" ] || abort "no EXE provided"

info "Preparing filesystem and environment..."

export KUBECONFIG_ROOT="$HOME/.kube"
mkdir -p "$KUBECONFIG_ROOT"

export TEST_OUTPUT_DIR="$HOME"/testoutput
mkdir -p "$TEST_OUTPUT_DIR"

info "Start time inside runner: $(date)"

function run_tests() {

  local section_name="$1"
  local prefix="$2"

  local test_files=("$CURR_DIR/$prefix"*".sh")

  section "$section_name ($CURR_DIR/$prefix*.sh)"

  num_total_tests=${#test_files[@]}

  local included_tests=()
  local excluded_tests=()

  for i in "${test_files[@]}"; do

    local base
    base=$(basename "$i" .sh)
    local skip=false
    local included=false
    local excluded=false

    # prepare to skip test, if it's in the exclusion list
    for excludetest in "${E2E_EXCLUDE[@]}"; do
      [[ "$excludetest" =~ (^| )${base}($| ) ]] && excluded=true
    done

    # (re-)add test to list, if it's on inclusion list
    for includetest in "${E2E_INCLUDE[@]}"; do
      [[ "$includetest" =~ (^| )${base}($| ) ]] && included=true
    done

    if [[ -z "${E2E_INCLUDE}" ]]; then # no explicit include list given
      if $excluded; then               # test is on explicit exclude list
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
      excluded_tests+=("$i")
    else
      included_tests+=("$i")
    fi
  done

  local num_included_tests=${#included_tests[@]}
  local num_excluded_tests=${#excluded_tests[@]}

  #
  # Run Tests
  #
  local max_batch_size=$E2E_PARALLEL
  local current_batch_size=0
  local current_batch_number=1
  local total_batch_number=$(((num_included_tests + (max_batch_size - 1)) / max_batch_size))

  info "Running $num_included_tests tests in $total_batch_number batches à max. $max_batch_size tests."

  info "Starting test batch #$current_batch_number/$total_batch_number..."

  for t in "${included_tests[@]}"; do

    if [[ current_batch_size -eq max_batch_size ]]; then
      info "Waiting for test batch #$current_batch_number/$total_batch_number to finish..."
      current_batch_size=0
      wait
      ((current_batch_number = current_batch_number + 1))
      info "Starting test batch #$current_batch_number/$total_batch_number..."
    fi

    local base
    base=$(basename "$t" .sh)
    highlight "***** Running $base *****"
    ((current_batch_size = current_batch_size + 1))
    "$t" &
  done

  wait

  # Output logs of failed tests
  local failed_logs=("$TEST_OUTPUT_DIR/$prefix"*".failed")
  local num_failed_tests=${#failed_logs[@]}
  if [[ num_failed_tests -gt 0 ]]; then
    warn "FAILED LOGS: ${failed_logs[*]}"
    for f in "${failed_logs[@]}"; do
      info "FAILED -> $f"
      local base
      base=$(basename "$f" ".log.failed")
      log "${RED}***************************************************${END}"
      log "${RED}*** Failed Test ${base%%.*}${END}"
      log "${RED}***************************************************${END}"
      cat "$f"
    done
  fi

  # Info Output about Test Results
  info "FINISHED $section_name${END}
  > ${WHT}Total:\t$num_total_tests${END}
  > ${BLU}Run:\t$num_included_tests${END}
  > ${YEL}Skipped:\t$num_excluded_tests${END}
  > ${GRN}Passed:\t$((num_included_tests - num_failed_tests))${END}
  > ${RED}Failed:\t$num_failed_tests${END}"

  # Error out if we encountered any failed test
  if [[ num_failed_tests -gt 0 ]]; then
    warn "Failed $section_name: $num_failed_tests"
    exit 1
  fi
}

###############
# Basic Tests #
###############

run_tests "BASIC TESTS" "test_"

###############
# Extra Tests #
###############
if [[ -n "$E2E_EXTRA" ]]; then
  run_tests "EXTRA TESTS" "extra_test_"
else
  info "NOT running EXTRA tests, please set E2E_EXTRA=1 if you wish to do so"
fi

exit 0
