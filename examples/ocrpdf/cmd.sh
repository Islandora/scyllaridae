#!/usr/bin/env bash

set -eou pipefail

TMP_DIR=$(mktemp -d)

cd "$TMP_DIR"

# split pdf into PNG files
magick - page-%d.png > /dev/null 2>&1

# add OCR to each PNG
for i in page-*.png; do
  tesseract "$i" "${i%.png}" --dpi 300 pdf > /dev/null 2>&1
done

# put the PDF back together
pdfunite page-*.pdf output.pdf > /dev/null 2>&1

# make sure the PDF is legit
pdfinfo output.pdf > /dev/null || exit 1

# print the results to stdout
cat output.pdf

# cleanup
cd /app
rm -rf "$TMP_DIR"
