#!/bin/sh
set -e

#####################################################
# Script to run both services in a single container #
#                                                   #
# This script is used by the all-in-one layer.      #
#####################################################

/dockerfiles-refresh \
    --data-file /data/dockerfiles.metadata.json \
    --interval "${REFRESH_INTERVAL:-1h}" \
    --log-level info &

exec /dockerfiles-dashboard
