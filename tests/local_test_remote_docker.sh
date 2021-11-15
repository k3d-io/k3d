#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }

# shellcheck source=./common.sh
source "$CURR_DIR/common.sh"

export CURRENT_STAGE="local | remote_docker"


info "Starting dind with TLS (sleeping for 10s to give it time to get ready)"
docker run -d -p 3376:2376 -e DOCKER_TLS_CERTDIR=/certs -v /tmp/dockercerts:/certs --privileged --rm --name k3dlocaltestdindsec docker:20.10-dind
sleep 10

info "Setting Docker Context"
docker context create k3dlocaltestdindsec --description "dind local secure" --docker "host=tcp://127.0.0.1:3376,ca=/tmp/dockercerts/client/ca.pem,cert=/tmp/dockercerts/client/cert.pem,key=/tmp/dockercerts/client/key.pem"
docker context use k3dlocaltestdindsec
docker context list

info "Running k3d"
k3d cluster create test1
k3d cluster list

info "Switching to default context"
docker context list
docker ps
docker context use default
docker ps

info "Checking DOCKER_TLS env var based setting"
export DOCKER_HOST=tcp://127.0.0.1:3376
export DOCKER_TLS_VERIFY=1
export DOCKER_CERT_PATH=/tmp/dockercerts/client

docker context list
docker ps
k3d cluster create test2
k3d cluster list
docker ps

info "Cleaning up"
unset DOCKER_HOST
unset DOCKER_TLS_VERIFY
unset DOCKER_CERT_PATH
k3d cluster rm -a
docker context use default
docker rm -f k3dlocaltestdindsec
docker context rm k3dlocaltestdindsec

info ">>> DONE <<<"
