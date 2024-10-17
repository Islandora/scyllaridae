#!/usr/bin/env bash

set -eou pipefail

TMP_DIR=$(mktemp -d)
I=0

# iterate over all images in the IIIF manifest
curl -s "$1/book-manifest" | jq -r '.sequences[0].canvases[].images[0].resource."@id"' | while read -r URL; do
  # resize image to max 1000px width
  curl -s "$URL" | convert -[0] -resize 1000x\> "$TMP_DIR/img_$I" > /dev/null 2>&1

  # make an OCR'd PDF from the image
  tesseract "$TMP_DIR/img_$I" "$TMP_DIR/img_$I" pdf > /dev/null 2>&1

  I="$(( I + 1))"
done

# Make the node title the title of the PDF
TITLE=$(curl -L "$1?_format=json" | jq -r '.title[0].value')
echo "[ /Title ($TITLE)/DOCINFO pdfmark" >  "$TMP_DIR/metadata.txt"

mapfile -t FILES < <(ls -rt "$TMP_DIR"/img_*.pdf)
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
