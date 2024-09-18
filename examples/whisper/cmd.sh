#!/usr/bin/env bash

# take input from stdin and print to stdout

set -eou pipefail

input_temp=$(mktemp /tmp/whisper-input-XXXXXX)

cat > "$input_temp"

libreoffice --headless --convert-to pdf "$input_temp" > /dev/null 2>&1

/app/main \
  -m /app/models/ggml-base.en.bin \
  --output-vtt \
  -f "$input_temp" 2>&1

rm "$input_temp"
