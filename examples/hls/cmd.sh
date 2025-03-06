#!/usr/bin/env bash

set -eou pipefail

SOURCE_MIMETYPE="$1"
NODE_URL="$2"
NID=$(basename "$NODE_URL")
TMP_DIR=$(mktemp -d)

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

ffmpeg \
  -f "$SOURCE_MIMETYPE" \
  -i - \
  -profile:v \
  baseline \
  -level 3.0 \
  -s 640x360 \
  -start_number 0 \
  -hls_time 10 \
  -hls_list_size 0 \
  -f hls \
  -b:v 800k \
  -maxrate 800k \
  -bufsize 1200k \
  -b:a 96k "$TMP_DIR/$NID.m3u8" > /dev/null 2>&1

if [ ! -f  "$TMP_DIR/$NID.m3u8" ]; then
  exit 1
fi

# shellcheck disable=SC2046
tar -czf "$TMP_DIR/hls.tar.gz" -C "$TMP_DIR" "./$NID.m3u8" $(cd "$TMP_DIR" ; echo ./*.ts)

BASE_URL=$(dirname "$NODE_URL" | xargs dirname)
TID=$(curl -s "$BASE_URL/term_from_term_name?vocab=islandora_media_use&name=Service+File&_format=json" | jq '.[0].tid[0].value')

curl -s \
  -H "Authorization: $SCYLLARIDAE_AUTH" \
  -H "Content-Type: application/gzip" \
  -H "Content-Location: private://derivatives/hls/$NID/hls.tar.gz" \
  -T "$TMP_DIR/hls.tar.gz" \
  "$NODE_URL/media/file/$TID"
