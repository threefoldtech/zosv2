#!/bin/bash
set -euo pipefail

# keepalived_version="1:1.3.9-1ubuntu0.18.04.2"

# echo "Installing keepalived ${keepalived_version}..."
# apt-get install -y --no-install-recommends "keepalived=${keepalived_version}"
# apt-mark hold keepalived

priority=100

# generate configuration file
cat <<EOF >/etc/keepalived/keepalived.conf
vrrp_instance VI_1 {
    interface ${NODE_NETWORK_TYPE}
    virtual_router_id 1
    priority ${priority}
    advert_int 1
    nopreempt
    authentication {
        auth_type AH
        auth_pass kubernetes
    }
    virtual_ipaddress {
        ${KUBE_MASTER_VIP}
    }
}
EOF

# enable and start keepalived
