#!/usr/bin/env bash

set -eou pipefail

if [ "$#" -ne 4 ]; then
  echo "Usage: $0 NODE-JSON-URL ORIGINAL-PDF-URL FILE-UPLOAD-URI DESTINATION-URI"
  exit 1
fi

NODE_JSON_URL="$1"
ORIGINAL_PDF_URL="$2"
FILE_UPLOAD_URI="$3"
DESTINATION_URI="$4"
TMP_DIR=$(mktemp -d)

# get the node JSON export
curl -L -s -o "$TMP_DIR/node.json" "${NODE_JSON_URL}?_format=json"
NODE_JSON=$(cat "$TMP_DIR/node.json")

# extract the title and citation from the node JSON
echo "$NODE_JSON" | jq -r '.field_full_title[0].value' > "$TMP_DIR/title.html"
echo "$NODE_JSON" | jq -r .citation[0].value > "$TMP_DIR/citation.html"

# The contents could contain MathML and other non-standard unicode characters
# so have pandoc convert them to LaTex
pandoc "$TMP_DIR/title.html" -o "$TMP_DIR/title-latex.tex"
pandoc "$TMP_DIR/citation.html" -o "$TMP_DIR/citation-latex.tex"

# replace new lines with a space
# and put the file in the location our main coverpage.tex will insert from
tr '\n' ' ' < "$TMP_DIR/title-latex.tex" > "$TMP_DIR/title.tex"
tr '\n' ' ' < "$TMP_DIR/citation-latex.tex" > "$TMP_DIR/citation.tex"

## Now generate the coverpage PDF

TMP_FILE="$TMP_DIR/coverpage.tex"
PDF_FILE="$TMP_DIR/coverpage.pdf"
MERGED_PDF="$TMP_DIR/merged.pdf"
EXISTING_PDF="$TMP_DIR/existing.pdf"

cp coverpage.tex "$TMP_FILE"

# Generate the cover page from LaTex to PDF
xelatex -output-directory="$TMP_DIR" "$TMP_FILE" > /dev/null

# Download the original PDF
curl -L -s -o "${EXISTING_PDF}" "$ORIGINAL_PDF_URL"

# Merge the cover page with the existing PDF using ghostscript
gs -dBATCH -dNOPAUSE -q -sDEVICE=pdfwrite -sOutputFile="${MERGED_PDF}" "$PDF_FILE" "$EXISTING_PDF"

# Upload the merged PDF
curl -X PUT \
  --header "Authorization: $SCYLLARIDAE_AUTH" \
  --header "Content-Location: $FILE_UPLOAD_URI" \
  --header "Content-Type: application/pdf" \
  --upload-file "$MERGED_PDF" \
  "$DESTINATION_URI"

# Cleanup
rm -r "$TMP_DIR" || echo "Could not cleanup temporary files"
