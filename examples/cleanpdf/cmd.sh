#!/usr/bin/env bash

set -eou pipefail

PDF=$(mktemp)

cleanup() {
  rm -rf "$PDF"
}
trap cleanup EXIT

cat > "$PDF"

qpdf --replace-input --flatten-annotations=all "$PDF" > /dev/null 2>&1

cat "$PDF"
