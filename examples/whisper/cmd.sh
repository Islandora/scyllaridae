#!/usr/bin/env bash

# take input from stdin and print to stdout

set -eou pipefail

BASE_URL=$(echo "$1" | xargs dirname)
input_temp=$(mktemp /tmp/whisper-input-XXXXXX)
output_file="${input_temp}_16khz.wav"

# replace relative *.ts URLs with the absolute URL to them
cat | sed 's|^\([^#].*\)|'"$BASE_URL"'/\1|' \
  | ffmpeg -protocol_whitelist https,fd,tls,tcp -f hls -i - -vn -acodec pcm_s16le -ar 16000 -ac 2 "$output_file"

# generate the VTT file
/app/main \
  -m /app/models/ggml-medium.en.bin \
  --output-vtt \
  -f "$output_file" \
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
