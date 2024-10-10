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

mapfile -t FILES < <(ls -rt "$TMP_DIR"/img_*.pdf)
gs -dBATCH \
  -dNOPAUSE \
  -dQUIET \
  -sDEVICE=pdfwrite \
  -dPDFA \
  -dNOOUTERSAVE \
  -dAutoRotatePages=/None \
  -sOutputFile="$TMP_DIR/ocr.pdf" \
  "${FILES[@]}"

cat "$TMP_DIR/ocr.pdf"
rm -rf "$TMP_DIR"
