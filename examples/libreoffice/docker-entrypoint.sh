#!/usr/bin/env bash

# take input from stdin and print to stdout

set -eou pipefail

input_temp=$(mktemp /tmp/libreoffice-input-XXXXXX)

cat > "$input_temp"

libreoffice --headless --convert-to pdf "$input_temp" > /dev/null 2>&1

PDF="/app/$(basename "$input_temp").pdf"
cat "$PDF"

rm "$input_temp" "$PDF"
