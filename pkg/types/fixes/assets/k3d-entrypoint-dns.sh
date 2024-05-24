#!/bin/sh

# DISCLAIMER
# Heavily inspired by / copied from https://github.com/kubernetes-sigs/kind/pull/1414/files#diff-3c55751d83af635109cece495ee2ff38206764a8b95f4cb8f11fc08a5c0ea8dc
# Apache 2.0 License (Kubernetes Authors): https://github.com/kubernetes-sigs/kind/blob/9222508298c50ce8c5ba1f364f37307e81ba915e/LICENSE

set -o errexit
set -o nounset

docker_dns="127.0.0.11"

gateway="GATEWAY_IP" # replaced within k3d Go code

echo "[$(date -Iseconds)] [DNS Fix] Use the detected Gateway IP $gateway instead of Docker's embedded DNS ($docker_dns)"

# Change iptables rules added by Docker to route traffic to out Gateway IP instead of Docker's embedded DNS
echo "[$(date -Iseconds)] [DNS Fix] > Changing iptables rules ..."
iptables-save \
  | sed \
    -e "s/-d ${docker_dns}/-d ${gateway}/g" \
    -e 's/-A OUTPUT \(.*\) -j DOCKER_OUTPUT/\0\n-A PREROUTING \1 -j DOCKER_OUTPUT/' \
    -e "s/--to-source :53/--to-source ${gateway}:53/g"\
  | iptables-restore

# Update resolv.conf to use the Gateway IP if needed: this will also make CoreDNS use it via k3s' default `forward . /etc/resolv.conf` rule in the CoreDNS config
if grep -q "${docker_dns}" /etc/resolv.conf; then
  echo "[$(date -Iseconds)] [DNS Fix] > Replacing IP in /etc/resolv.conf ..."
  cp /etc/resolv.conf /etc/resolv.conf.original
  sed -e "s/${docker_dns}/${gateway}/g" /etc/resolv.conf.original >/etc/resolv.conf
fi

echo "[$(date -Iseconds)] [DNS Fix] Done"
