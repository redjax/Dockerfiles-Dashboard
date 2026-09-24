#!/bin/sh
set -e

#####################################################
# Script to run both services in a single container #
#                                                   #
# This script is used by the all-in-one layer.      #
#####################################################

mkdir -p /data

/dockerfiles-refresh \
    --data-file "${DATA_FILE}" \
    --interval "${REFRESH_INTERVAL:-1h}" \
    --log-level info &

exec /dockerfiles-dashboard
