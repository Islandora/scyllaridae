#!/usr/bin/env bash

# use sed to find/replace placeholders for nginx-ingress

set -eou pipefail

TARGET="ingress.yaml"
if [ $# -eq 1 ]; then
  if [ ! -f "$1" ]; then
    echo "$1 doesn't exit"
    exit 1
  fi

  TARGET="$1"
fi

sed -e "s|__DOMAIN__|$DOMAIN|" \
    -e "s|__KUBE_TLS_SECRET__|$KUBE_TLS_SECRET|" \
    "$@" \
| kubectl apply -f -
