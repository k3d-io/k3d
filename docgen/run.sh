#!/bin/bash

REPLACE_PLACEHOLDER="/PATH/TO/YOUR/REPO/DIRECTORY"

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

REPO_DIR=${CURR_DIR%"/docgen"}

echo "$REPO_DIR"

sed -i "s%$REPLACE_PLACEHOLDER%$REPO_DIR%" "$CURR_DIR/go.mod"

go mod tidy

go mod vendor

go run ./main.go
