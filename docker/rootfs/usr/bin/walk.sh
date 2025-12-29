#!/bin/bash

set -euxo pipefail

# Trap signals and exit
trap "exit 0" SIGHUP SIGINT SIGTERM

source /etc/environment
exec /usr/bin/mesh-manager walk
