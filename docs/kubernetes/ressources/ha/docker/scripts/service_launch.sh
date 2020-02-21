#!/bin/bash
set -Eeo pipefail

service haproxy start
service keepalived start
echo "done bro"
