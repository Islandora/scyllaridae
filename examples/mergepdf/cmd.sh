#!/usr/bin/env bash

set -eou pipefail

TMP_DIR=$(mktemp -d)
I=0
MAX_THREADS=${MAX_THREADS:-5}
PIDS=()

# iterate over all images in the IIIF manifest
URLS=$(curl -s "$1/book-manifest" | jq -r '.sequences[0].canvases[].images[0].resource."@id"' | awk -F '/' '{print $7}'|sed -e 's/%2F/\//g' -e 's/%3A/:/g')
while read -r URL; do
  # If we have reached the max thread limit, wait for any one job to finish
  if [ "${#PIDS[@]}" -ge "$MAX_THREADS" ]; then
    wait -n
    NEW_PIDS=()
    for pid in "${PIDS[@]}"; do
      if kill -0 "$pid" 2>/dev/null; then
        NEW_PIDS+=("$pid")
      fi
    done
    PIDS=("${NEW_PIDS[@]}")
  fi

  # Run each job in the background
  (
    # download and resize image to max 1000px width
    curl -s "$URL" | magick -[0] -resize 1000x\> "$TMP_DIR/img_$I" || curl -s "$URL" | magick - -resize 1000x\> "$TMP_DIR/img_$I" > /dev/null 2>&1
    # make an OCR'd PDF from the image
    tesseract "$TMP_DIR/img_$I" "$TMP_DIR/img_$I" pdf > /dev/null 2>&1
    rm "$TMP_DIR/img_$I"
  ) &
  PIDS+=("$!")
  I="$(( I + 1))"
done <<< "$URLS"

FILES=()
for index in $(seq 0 $((I - 1))); do
  FILES+=("$TMP_DIR/img_${index}.pdf")
done

wait

# Make the node title the title of the PDF
TITLE=$(curl -L "$1?_format=json" | jq -r '.title[0].value')
echo "[ /Title ($TITLE)/DOCINFO pdfmark" >  "$TMP_DIR/metadata.txt"

gs -dBATCH \
  -dNOPAUSE \
  -dQUIET \
  -sDEVICE=pdfwrite \
  -dPDFA \
  -dNOOUTERSAVE \
  -dAutoRotatePages=/None \
  -sOutputFile="$TMP_DIR/ocr.pdf" \
  "${FILES[@]}" \
  "$TMP_DIR/metadata.txt"

# Instead of printing the PDF
# PUT it to the endpoint
NID=$(basename "$1")
BASE_URL=$(dirname "$1" | xargs dirname)
TID=$(curl "$BASE_URL/term_from_term_name?vocab=islandora_media_use&name=Original+File&_format=json" | jq '.[0].tid[0].value')
curl \
  -H "Authorization: $SCYLLARIDAE_AUTH" \
  -H "Content-Type: application/pdf" \
  -H "Content-Location: private://derivatives/pc/pdf/$NID.pdf" \
  -T "$TMP_DIR/ocr.pdf" \
  "$1/media/document/$TID"

rm -rf "$TMP_DIR"

echo "OK"
