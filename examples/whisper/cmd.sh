#!/usr/bin/env bash

# take input from stdin and print to stdout

set -eou pipefail

if [ ! -f /app/models/ggml-medium.en.bin ]; then
  bash ./models/download-ggml-model.sh medium.en > /dev/null 2>&1
fi

# take stdin and buffer it into a temp file
input_temp=$(mktemp /tmp/whisper-input-XXXXXX)
cat > "$input_temp"

# make sure we have a 16kHz WAV file
output_file="${input_temp}_16khz.wav"
ffmpeg -i "$input_temp" -ar 16000 -ac 1 "$output_file"

# generate the VTT file
/app/main \
  -m /app/models/ggml-medium.en.bin \
  --output-vtt \
  -f "$input_temp" \
  --output-file "$input_temp" > /dev/null 2>&1 || true

# make sure a VTT file was created
STATUS=$(head -1  "$input_temp.vtt" | grep WEBVTT || echo "FAIL")
if [ "$STATUS" != "FAIL" ]; then
  cat "$input_temp.vtt"
fi

rm "$input_temp" "$input_temp.vtt" "$output_file"

if [ "$STATUS" == "FAIL" ]; then
  exit 1
fi
