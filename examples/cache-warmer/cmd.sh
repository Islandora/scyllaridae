#!/usr/bin/env bash

set -eou pipefail

mygrep() {
  # mac OS needs ggrep for the -P flag
  if command -v ggrep &>/dev/null; then
    ggrep "$@"
  else
    grep "$@"
  fi
}

# how many cURL commands to run in parallel
PARALLEL_EXECUTIONS=3

# Base URL of the sitemap.xml file
BASE_URL="https://$DOMAIN/sitemap.xml"
PAGE=1

while true; do
  NEXT_PAGE_URL="$BASE_URL?page=$PAGE"
  STATUS=$(curl -w '%{http_code}' \
    --silent \
    -o links.xml \
    "${NEXT_PAGE_URL}")

  if [ "${STATUS}" -eq 200 ]; then
    mapfile -t URLS < <(mygrep -oP '<loc>\K[^<]+' links.xml)
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
        curl --silent -o /dev/null "${URL}" &
        job_ids+=($!)
      done

      for job_id in "${job_ids[@]}"; do
        wait "$job_id" || echo "One job failed, but continuing anyway"
      done
    done

    PAGE=$((PAGE + 1))
  else
    break
  fi
done

rm -f links.xml

curl -v "https://$DOMAIN/api/v1/paged-content" > pc.json

mapfile -t NIDS < <(jq -r '.[]' pc.json)
for NID in "${NIDS[@]}"; do
  echo "Processing: $NID"
  curl -s -o /dev/null "https://$DOMAIN/node/$NID/book-manifest?cache-warmer=1"
done

rm -f pc.json
