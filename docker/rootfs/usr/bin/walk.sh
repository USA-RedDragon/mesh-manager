#!/bin/bash

set -euxo pipefail

# Trap signals and exit
trap "exit 0" SIGHUP SIGINT SIGTERM

export $(cat /etc/environment | xargs)
exec /usr/bin/mesh-manager walk
