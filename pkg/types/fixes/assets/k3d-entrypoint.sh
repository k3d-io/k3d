#!/bin/sh

set -o errexit
set -o nounset

LOGFILE="/var/log/k3d-entrypoints_$(date "+%y%m%d%H%M%S").log"

touch "$LOGFILE"

echo "[$(date -Iseconds)] Running k3d entrypoints..." >> "$LOGFILE"

for entrypoint in /bin/k3d-entrypoint-*.sh ; do
  echo "[$(date -Iseconds)] Running $entrypoint"  >> "$LOGFILE"
  "$entrypoint"  >> "$LOGFILE" 2>&1 || exit 1
done

echo "[$(date -Iseconds)] Finished k3d entrypoint scripts!" >> "$LOGFILE"

# Capture the k3s subcommand ("server" or "agent") before starting k3s.
# We need this in a variable because $1 inside functions refers to the
# function's arguments, not the script's.
K3S_ROLE="${1:-}"

/bin/k3s "$@" &
k3s_pid=$!

# Only server nodes have a kubeconfig at /etc/rancher/k3s/k3s.yaml that
# grants kubectl API access. Agent nodes have no usable kubeconfig, so
# running kubectl commands on them fails in an infinite retry loop (#1420,
# #1535). Drain/uncordon is also semantically a server-side scheduling
# operation — agents don't need to drain themselves.
#
# We match "server" explicitly rather than excluding "agent" so that any
# unexpected value (manual docker run, future k3s subcommands) falls
# through to the safe default: SIGTERM forwarding only.
if [ "$K3S_ROLE" = "server" ]; then
  # shellcheck disable=SC3028
  until kubectl uncordon "$HOSTNAME"; do sleep 3; done
fi

# shellcheck disable=SC3028
cleanup() {
  if [ "$K3S_ROLE" = "server" ]; then
    echo "Draining node..."
    kubectl drain "$HOSTNAME" --force --delete-emptydir-data --ignore-daemonsets --disable-eviction
  fi
  echo "Sending SIGTERM to k3s..."
  kill -15 $k3s_pid
  echo "Waiting for k3s to close..."
  wait $k3s_pid
  echo "Bye!"
}

# shellcheck disable=SC3048
trap cleanup SIGTERM SIGINT SIGQUIT SIGHUP

wait $k3s_pid
echo "k3d cleanup finished! Bye!"
