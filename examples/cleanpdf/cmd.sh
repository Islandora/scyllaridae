#!/usr/bin/env bash

set -eou pipefail

PDF=$(mktemp)

cat > "$PDF"

qpdf --replace-input --flatten-annotations=all "$PDF" > /dev/null 2>&1

cat "$PDF"

rm "$PDF"
