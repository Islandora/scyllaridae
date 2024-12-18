#!/usr/bin/env bash

set -eou pipefail

export LOCK_FILE="/tmp/scyllaridae-cache.lock"

# how many cURL commands to run in parallel for /node/\d+
if [ ! -v NODE_PARALLEL_EXECUTIONS ] || [ "$NODE_PARALLEL_EXECUTIONS" = "" ]; then
  NODE_PARALLEL_EXECUTIONS=5
fi

# how many cURL commands to run in parallel for IIIF manifests
if [ ! -v IIIF_PARALLEL_EXECUTIONS ] || [ "$IIIF_PARALLEL_EXECUTIONS" = "" ]; then
  IIIF_PARALLEL_EXECUTIONS=3
fi


handle_error() {
  rm -f "$LOCK_FILE"
  exit 1
}
trap 'handle_error' ERR

# curl wrapper function so on 302 we can forward the cache-warmer paramater
process_url() {
  local URL="$1"
  local COUNT=0
  echo "Crawling: $URL"
  REDIRECT_URL=$(curl -w "%{redirect_url}" --silent -o /dev/null "$URL")
  while [ "$REDIRECT_URL" != "" ]; do
    REDIRECT_URL=$(curl -w "%{redirect_url}" --silent -o /dev/null "$REDIRECT_URL?cache-warmer=1")
    COUNT=$((COUNT + 1))
    if [ "$COUNT" -gt 5 ]; then
      break
    fi
  done
}

# if we just need to warm the cache for a single node, do that then bail
if [ "$#" -eq 1 ] && [ "$1" != "all" ]; then
  process_url "$1?cache-warmer=1"
  exit 0
fi

# otherwise we're warming the entire site's cache

if [ -f "$LOCK_FILE" ]; then
  # TODO: we need a lock mechanism in scyllardiae that can kill running processes
  # but for now we can just gate it here
  echo "Cache warming is already taking place"
  exit 0
fi

touch "$LOCK_FILE"

# Warm everything in the sitemap
BASE_URL="$DRUPAL_URL/sitemap.xml"
PAGE=1

while true; do
  NEXT_PAGE_URL="$BASE_URL?page=$PAGE"
  STATUS=$(curl -w '%{http_code}' \
    --silent \
    -o links.xml \
    "${NEXT_PAGE_URL}")

  if [ "${STATUS}" -eq 200 ]; then
    mapfile -t URLS < <(grep -oP '<loc>\K[^<]+' links.xml)
    while [ "${#URLS[@]}" -gt 0 ]; do
      for ((i = 0; i < NODE_PARALLEL_EXECUTIONS; i++)); do
        array_length=${#URLS[@]}
        if [ "$array_length" -gt 0 ]; then
          URL="${URLS[$((array_length-1))]}"
          unset "URLS[$((array_length-1))]"
        else
          break
        fi
        process_url "$URL?cache-warmer=1" &
        job_ids+=($!)
      done

      for job_id in "${job_ids[@]}"; do
        wait "$job_id" || echo "One job failed, but continuing anyway"
      done
    done

    PAGE=$((PAGE + 1))
    if [ "$PAGE" -gt 100 ]; then
      break
    fi
  else
    break
  fi
done

rm -f links.xml

# now that the sitemap is warm, get all the IIIF paged content manifests warm
curl -s "$DRUPAL_URL/api/v1/paged-content" > pc.json
mapfile -t NIDS < <(jq -r '.[]' pc.json)
for NID in "${NIDS[@]}"; do
  for ((i = 0; i < IIIF_PARALLEL_EXECUTIONS; i++)); do
    array_length=${#NIDS[@]}
    if [ "$array_length" -gt 0 ]; then
      NID="${NIDS[$((array_length-1))]}"
      unset "NIDS[$((array_length-1))]"
    else
      break
    fi
    echo "Crawling: $DRUPAL_URL/node/$NID/book-manifest"
    curl -s -o /dev/null "$DRUPAL_URL/node/$NID/book-manifest" &
    job_ids+=($!)
  done

  for job_id in "${job_ids[@]}"; do
    wait "$job_id" || echo "One job failed, but continuing anyway"
  done
done

rm -f pc.json

rm "$LOCK_FILE"
