#!/bin/bash
set -euo pipefail

# haproxy_version="1.8.24-1ppa1~bionic"
# export RUNLEVEL=1
# echo "Installing haproxy ${haproxy_version}..."
# apt-get install -y --no-install-recommends "haproxy=${haproxy_version}"
# apt-mark hold haproxy

# create a configuration file for all other master hosts
cat <<EOF >>/etc/haproxy/haproxy.cfg

frontend kube-api
  bind 0.0.0.0:8443
  bind 127.0.0.1:8443
  mode tcp
  option tcplog
  timeout client 4h
  default_backend kube-api

backend kube-api
  mode tcp
  option tcp-check
  timeout server 4h
  balance roundrobin
  default-server inter 10s downinter 5s rise 2 fall 2 slowstart 60s maxconn 250 maxqueue 256 weight 100
  server kube-master-01 ${KUBE_MASTER_IP_01}:6443 check
  server kube-master-02 ${KUBE_MASTER_IP_02}:6443 check
  server kube-master-03 ${KUBE_MASTER_IP_03}:6443 check
EOF

# reload after new configuration file has been updated
# curl -sSLk "https://192.168.168.100:8443"
