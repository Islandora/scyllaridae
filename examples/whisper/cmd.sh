#!/usr/bin/env bash

# take input from stdin and print to stdout

set -eou pipefail

if [ ! -f /app/models/ggml-medium.en.bin ]; then
  bash ./models/download-ggml-model.sh medium.en
fi

input_temp=$(mktemp /tmp/whisper-input-XXXXXX)

cat > "$input_temp"

/app/main \
  -m /app/models/ggml-medium.en.bin \
  --output-vtt \
  -f "$input_temp" \
  --output-file "$input_temp" > /dev/null 2>&1

cat "$input_temp.vtt"

rm "$input_temp" "$input_temp.vtt"
