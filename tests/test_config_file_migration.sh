#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

### Step Setup ###
# Redirect all stdout/stderr output to logfile
LOG_FILE="$TEST_OUTPUT_DIR/$( basename "${BASH_SOURCE[0]}" ).log"
exec >${LOG_FILE} 2>&1
export LOG_FILE

# use a kubeconfig file specific to this test
KUBECONFIG="$KUBECONFIG_ROOT/$( basename "${BASH_SOURCE[0]}" ).yaml"
export KUBECONFIG
### Step Setup ###


export CURRENT_STAGE="Test | config-file-migration"



highlight "[START] ConfigMigrateTest"

tempdir=$(mktemp -d)
$EXE config migrate "$CURR_DIR/assets/config_test_simple_migration_v1alpha2.yaml" "$tempdir/expected.yaml" || failed "failed on $CURR_DIR/assets/config_test_simple_migration_v1alpha2.yaml"
$EXE config migrate "$CURR_DIR/assets/config_test_simple_migration_v1alpha3.yaml" "$tempdir/actual.yaml" || failed "failed on $CURR_DIR/assets/config_test_simple_migration_v1alpha3.yaml"
# TODO: migrate to v1alpha4
# TODO: migrate to v1alpha5

diff "$tempdir/actual.yaml" "$tempdir/expected.yaml" || failed "config migration failed" && passed "config migration succeeded"


highlight "[DONE] ConfigMigrateTest"

exit 0


