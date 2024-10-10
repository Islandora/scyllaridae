#!/usr/bin/env bash

set -eou pipefail

TMP_DIR=$(mktemp -d)
I=0

# iterate over all images in the IIIF manifest
curl -s "$1/book-manifest" | jq -r '.sequences[0].canvases[].images[0].resource."@id"' | while read -r URL; do
  # resize image to max 1500px width
  curl -s "$URL" | convert -[0] -resize 1500x\> "$TMP_DIR/img_$I"

  # make an OCR'd PDF from the image
  tesseract "$TMP_DIR/img_$I" "$TMP_DIR/img_$I" pdf

  I="$(( I + 1))"
done

gs -dBATCH \
  -dNOPAUSE \
  -dQUIET \
  -sDEVICE=pdfwrite \
  -dCompatibilityLevel=1.4 \
  -dPDFSETTINGS=/screen \
  -dPDFA \
  -sProcessColorModel=DeviceRGB \
  -dNOOUTERSAVE \
  -dAutoRotatePages=/None \
  -sPDFACompatibilityPolicy=1 \
  -sOutputFile="$TMP_DIR/ocr.pdf" \
  "$TMP_DIR/img_*.pdf"

cat "$TMP_DIR/ocr.pdf"
rm -rf "$TMP_DIR"
