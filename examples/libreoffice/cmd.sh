#!/usr/bin/env bash

# take input from stdin and print to stdout

set -eou pipefail

input_temp=$(mktemp /tmp/libreoffice-input-XXXXXX)
PDF="/app/$(basename "$input_temp").pdf"

cleanup() {
  rm -rf "$input_temp" "$PDF"
}
trap cleanup EXIT

cat > "$input_temp"

libreoffice --headless --convert-to pdf "$input_temp" > /dev/null 2>&1

cat "$PDF"
