#!/bin/bash

RED='\033[1;31m'
GRN='\033[1;32m'
YEL='\033[1;33m'
BLU='\033[1;34m'
WHT='\033[1;37m'
MGT='\033[1;95m'
CYA='\033[1;96m'
END='\033[0m'
BLOCK='\033[1;37m'

PATH=/usr/local/bin:$PATH
export PATH

log() { >&2 printf "$(date -u +"%Y-%m-%d %H:%M:%S %Z") [${CURRENT_STAGE:-undefined}] ${BLOCK}>>>${END} $1\n"; }

info() { log "${BLU}$1${END}"; }
highlight() { log "${MGT}$1${END}"; }

bye() {
  log "${BLU}$1... exiting${END}"
  exit 0
}

warn() { log "${RED}WARN: $1 ${END}"; }

abort() {
  log "${RED}FATAL: $1${END}"
  exit 1
}

command_exists() {
    command -v "$1" >/dev/null 2>&1
}

failed() {
  if [ -z "$1" ] ; then
    log "${RED}failed!!!${END}"
  else
    log "${RED}$1${END}"
  fi
  if [[ -n "$2" ]]; then
    mv "$2" "$2.failed"
  elif [[ -n "$LOG_FILE" ]]; then
    mv "$LOG_FILE" "$LOG_FILE.failed"
  fi
  abort "$CURRENT_STAGE: test failed"
  if [[ "$E2E_FAIL_FAST" == "true" ]]; then
    exit 1
  fi
}

passed() {
  if [ -z "$1" ] ; then
    log "${GRN}done!${END}"
  else
    log "${GRN}$1${END}"
  fi
}

section() {
  title_length=$((${#1}+4))
  log "$(printf "${CYA}#%.0s${END}" $(eval "echo {1.."$(($title_length))"}"); printf "\n";)"
  log "$(printf "${CYA}#${END} %s ${CYA}#${END}\n" "$1")"
  log "$(printf "${CYA}#%.0s${END}" $(eval "echo {1.."$(($title_length))"}"); printf "\n";)"
}

# checks that a URL is available, with an optional error message
check_url() {
  command_exists curl || abort "curl is not installed"
  curl -L --silent -k --output /dev/null --fail "$1"
}

# check_clusters verifies that given clusters are reachable
check_clusters() {
  [ -n "$EXE" ] || abort "EXE is not defined"
  for c in "$@" ; do
    if $EXE kubeconfig merge "$c" --kubeconfig-switch-context; then
      if kubectl cluster-info ; then
        passed "cluster $c is reachable"
      else
        warn "could not obtain cluster info for $c. Kubeconfig:\n$(kubectl config view)"
        docker ps -a
        return 1
      fi
    else
      warn "could not merge kubeconfig for $c."
      docker ps -a
      return 1
    fi
  done
  return 0
}

check_cluster_count() {
  expectedClusterCount=$1
  shift # all remaining args are clusternames
  actualClusterCount=$(LOG_LEVEL=warn $EXE cluster list --no-headers "$@" | wc -l) # this must always have a loglevel of <= warn or it will fail
  if [[ $actualClusterCount -ne $expectedClusterCount ]]; then
    failed "incorrect number of clusters available: $actualClusterCount != $expectedClusterCount"
    return 1
  fi
  return 0
}

# check_multi_node verifies that a cluster runs with an expected number of nodes
check_multi_node() {
  cluster=$1
  expectedNodeCount=$2
  $EXE kubeconfig merge "$cluster" --kubeconfig-switch-context
  nodeCount=$(kubectl get nodes -o=custom-columns=NAME:.metadata.name --no-headers | wc -l)
  if [[ $nodeCount -eq $expectedNodeCount ]]; then
    passed "cluster $cluster has $expectedNodeCount nodes, as expected"
  else
    warn "cluster $cluster has incorrect number of nodes: $nodeCount != $expectedNodeCount"
    kubectl get nodes
    docker ps -a
    return 1
  fi
  return 0
}

check_registry() {
  check_url "$REGISTRY/v2/_catalog"
}

check_volume_exists() {
  docker volume inspect "$1" >/dev/null 2>&1
}

check_cluster_token_exist() {
  [ -n "$EXE" ] || abort "EXE is not defined"
  $EXE cluster get "$1" --token | grep -q "TOKEN" >/dev/null 2>&1
}

wait_for_pod_running_by_label() {
  podname=$(kubectl get pod -l "$1" $([[ -n "$2" ]] && echo "--namespace $2") -o jsonpath='{.items[0].metadata.name}')
  wait_for_pod_running_by_name "$podname" "$2"
}

wait_for_pod_running_by_name() {
  while : ; do
    podstatus=$(kubectl get pod "$1" $([[ -n "$2" ]] && echo "--namespace $2") -o go-template='{{.status.phase}}')
    case "$podstatus" in
      "ErrImagePull" )
        echo "Pod $1 is NOT running: ErrImagePull"
        return 1
        ;;
      "ContainerCreating" )
        continue
        ;;
      "Pending" )
        continue
        ;;
      "Running" )
        echo "Pod $1 is Running"
        return 0
        ;;
      * )
        echo "Pod $1 is NOT running: Unknown status '$podstatus'"
        kubectl describe pod "$1" || kubectl get pods $([[ -n "$2" ]] && echo "--namespace $2")
        return 1
    esac
  done
}

wait_for_pod_exec() {
  # $1 = pod name
  # $2 = command
  # $3 = max. retries (default: 10)
  max_retries=$([[ -n "$3" ]] && echo "$3" || echo "10")
  for (( i=0; i<=max_retries; i++ )); do
    echo "Try #$i: 'kubectl exec $1 -- $2'"
    kubectl exec "$1" -- $2 > /dev/null && return 0
    sleep 1
  done
  echo "Command '$2' in pod '$1' did NOT return successfully in $max_retries tries"
  return 1
}

exec_in_node() {
  # $1 = container/node name
  # $2 = command
  docker exec "$1" sh -c "$2"
}

docker_assert_container_label() {
  # $1 = container/node name
  # $2 = label to assert
  docker inspect --format '{{ range $k, $v := .Config.Labels }}{{ printf "%s=%s\n" $k $v }}{{ end }}' "$1" | grep -qE "^$2$"
}

k3s_assert_node_label() {
  # $1 = node name
  # $2 = label to assert
  kubectl get node "$1" --output go-template='{{ range $k, $v := .metadata.labels }}{{ printf "%s=%s\n" $k $v }}{{ end }}' | grep -qE "^$2$"
}
