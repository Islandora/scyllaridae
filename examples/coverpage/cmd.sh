#!/usr/bin/env bash

set -eou pipefail

if [ "$#" -ne 1 ]; then
  echo "Usage: ./cmd.sh NODE-JSON-URL"
  exit 1
fi

NODE_JSON_URL="$1"
TMP_DIR=$(mktemp -d)

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

EXISTING_PDF="$TMP_DIR/existing.pdf"
cat > "${EXISTING_PDF}"

# get the node JSON export
curl -L -s -o "$TMP_DIR/node.json" -H "Authorization: $SCYLLARIDAE_AUTH" "${NODE_JSON_URL}?_format=json"
NODE_JSON=$(cat "$TMP_DIR/node.json")

# extract the title and citation from the node JSON
# instead of installing a perl module or python or php
# just do some basic html entity decoding with sed replacements
# so things like MathML can get rendered correctly
echo "$NODE_JSON" | jq -r '.field_full_title[0].value' | \
  sed -e 's/&lt;/</g' \
      -e 's/&gt;/>/g' \
      -e 's/&amp;/\&/g' \
      -e 's/&quot;/"/g' \
      -e 's/&apos;/'"'"'/g' \
      -e 's/< mml/<\/mml/g' > "$TMP_DIR/title.html"

echo "$NODE_JSON" | jq -r '.citation[0].value' | \
  sed -e 's/&lt;/</g' \
      -e 's/&gt;/>/g' \
      -e 's/&amp;/\&/g' \
      -e 's/&quot;/"/g' \
      -e 's/&apos;/'"'"'/g' \
      -e 's/< mml/<\/mml/g' > "$TMP_DIR/citation.html"

# The title and citation could contain MathML and other non-standard unicode characters
# so have pandoc convert them to LaTex
pandoc "$TMP_DIR/title.html" -o "$TMP_DIR/title-latex.tex" --lua-filter=/app/urldecode.lua > /dev/null 2>&1
pandoc "$TMP_DIR/citation.html" -o "$TMP_DIR/citation-latex.tex" --lua-filter=/app/urldecode.lua > /dev/null 2>&1

# replace new lines with a space
# and put the file in the location our main coverpage.tex will insert from
tr '\n' ' ' < "$TMP_DIR/title-latex.tex" > "$TMP_DIR/title.tex"
tr '\n' ' ' < "$TMP_DIR/citation-latex.tex" > "$TMP_DIR/citation.tex"

## Now generate the coverpage PDF

TMP_FILE="$TMP_DIR/coverpage.tex"
PDF_FILE="$TMP_DIR/coverpage.pdf"
MERGED_PDF="$TMP_DIR/merged.pdf"

cp coverpage.tex "$TMP_FILE"

# Generate the cover page from LaTex to PDF
xelatex -output-directory="$TMP_DIR" "$TMP_FILE" > /dev/null 2>&1

# Merge the cover page with the existing PDF using ghostscript
gs -dBATCH -dNOPAUSE -q -sDEVICE=pdfwrite -sOutputFile="${MERGED_PDF}" "$PDF_FILE" "$EXISTING_PDF" > /dev/null 2>&1

cat "$MERGED_PDF"

