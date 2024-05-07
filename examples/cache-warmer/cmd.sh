#!/usr/bin/env bash

set -eou pipefail

# how many cURL commands to run in parallel
PARALLEL_EXECUTIONS=3

# Base URL of the sitemap.xml file
BASE_URL="$DRUPAL_URL/sitemap.xml"
PAGE=1

process_url() {
  local URL="$1"
  local COUNT=0
  echo "Crawling: $URL"
  REDIRECT_URL=$(curl -w "%{redirect_url}" --silent -o /dev/null "$URL")
  while [ "$REDIRECT_URL" != "" ]; then
    REDIRECT_URL=$(curl -w "%{redirect_url}" --silent -o /dev/null "$REDIRECT_URL?cache-warmer=1")
    COUNT=$((COUNT + 1))
    if [ "$COUNT" -gt 5 ]; then
      break
    fi
  fi
}

while true; do
  NEXT_PAGE_URL="$BASE_URL?page=$PAGE"
  STATUS=$(curl -w '%{http_code}' \
    --silent \
    -o links.xml \
    "${NEXT_PAGE_URL}")

  if [ "${STATUS}" -eq 200 ]; then
    mapfile -t URLS < <(grep -oP '<loc>\K[^<]+' links.xml)
    while [ "${#URLS[@]}" -gt 0 ]; do
      for ((i = 0; i < PARALLEL_EXECUTIONS; i++)); do
        array_length=${#URLS[@]}
        if [ "$array_length" -gt 0 ]; then
          URL="${URLS[$((array_length-1))]}"
          unset "URLS[$((array_length-1))]"
        else
          break
        fi
        echo "Crawling: $URL"
        process_url "$URL?cache-warmer=1" &
        job_ids+=($!)
      done

      for job_id in "${job_ids[@]}"; do
        wait "$job_id" || echo "One job failed, but continuing anyway"
      done
    done

    PAGE=$((PAGE + 1))
    if [ PAGE -gt 100 ]; then
      break
    fi
  else
    break
  fi
done

rm -f links.xml

curl -v "$DRUPAL_URL/api/v1/paged-content" > pc.json

mapfile -t NIDS < <(jq -r '.[]' pc.json)
for NID in "${NIDS[@]}"; do
  echo "Processing: $NID"
  curl -s -o /dev/null "$DRUPAL_URL/node/$NID/book-manifest?cache-warmer=1"
done

rm -f pc.json
