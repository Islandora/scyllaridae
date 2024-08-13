#!/usr/bin/env bash

set -eou pipefail

TARGET="*.yaml"
if [ -n "$1" ]; then
  TARGET="$1"
fi

sed -e "s|__DOMAIN__|$DOMAIN|" \
    -e "s|__DOCKER_REPOSITORY__|$DOCKER_REPOSITORY|" \
    -e "s|__KUBE_TLS_SECRET__|$KUBE_TLS_SECRET|" \
    "$TARGET" \
| kubectl apply -f -
