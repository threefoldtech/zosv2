#!/bin/bash
set -Eeo pipefail

cd /home/scripts

# export KUBE_MASTER_VIP=192.168.168.100
# export KUBE_MASTER_IP_01=192.168.168.101
# export KUBE_MASTER_IP_02=192.168.168.102
# export KUBE_MASTER_IP_03=192.168.168.103
# export NODE_NETWORK_TYPE=eth0
./haproxy.sh
./keepalived.sh

# curl -sfL https://get.k3s.io | sh -
curl -fL https://github.com/rancher/k3s/releases/download/v1.17.3-rc1%2Bk3s1/k3s -o k3s
chmod +x k3s
mv k3s /usr/local/bin/k3s

echo "alias k='/usr/local/bin/k3s kubectl'" >>~/.bashrc
echo "Hello World! This is the Ubuntu Container!"
