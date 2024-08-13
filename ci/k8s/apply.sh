#!/usr/bin/env bash

set -eou pipefail

TARGET="*.yaml"
if [ $# -eq 1 ]; then
  if [ ! -f "$1" ]; then
    echo "$1 doesn't exit"
    exit 1
  fi

  TARGET="$1"
fi

# Use eval to expand the wildcard in the TARGET variable
eval "set -- $TARGET"

sed -e "s|__DOMAIN__|$DOMAIN|" \
    -e "s|__DOCKER_REPOSITORY__|$DOCKER_REPOSITORY|" \
    -e "s|__KUBE_TLS_SECRET__|$KUBE_TLS_SECRET|" \
    "$@" \
| kubectl apply -f -
