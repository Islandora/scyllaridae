#!/usr/bin/env bash

set -eou pipefail

if [ "$#" -ne 4 ]; then
  echo "Usage NODE-JSON-URL ORIGINAL-PDF-URL FILE-UPLOAD-URI DESTINATION-URI"
  exit 1
fi

NODE_JSON=$(curl -L -s "$1?_format=json")

TITLE=$(echo "$NODE_JSON" | jq -r .title[0].value)

# Make any URL in the citation an href
# and convert <i> tags to \textit
CITATION=$(echo "$NODE_JSON" | jq -r .citation[0].value | \
  sed -E 's|(https?://[a-zA-Z0-9./?=_-]+)([.,;!?])|\\\\href{\1}{\1}\2|g' | \
  sed -E 's|<i>([^<]+)</i>|\\\\textit{\1}|g')

TMP_FILE=$(mktemp)

sed -e "s|TITLE|${TITLE}|" \
    -e "s|CITATION|${CITATION}|" \
    coverpage.tex > "$TMP_FILE.tex"

pdflatex "$TMP_FILE.tex" > /dev/null

COVERPAGE_FILE="$(basename "$TMP_FILE").pdf"
ORIGINAL_PDF="${TMP_FILE}.pdf"
MERGED_PDF=$(mktemp)

# download the original PDF
curl -L -s -o "${ORIGINAL_PDF}" "$2"

gs -dBATCH -dNOPAUSE -q -sDEVICE=pdfwrite -sOutputFile="${MERGED_PDF}" "${COVERPAGE_FILE}" "${ORIGINAL_PDF}"

rm "$TMP_FILE" "$TMP_FILE.tex" "$(basename "$TMP_FILE").*"

curl -XPUT \
  --header "Authorization: $SCYLLARIDAE_AUTH" \
  --header "Content-Location: $3" \
  --header "Content-Type: application/pdf" \
  --upload-file "$MERGED_PDF" \
  "$4"
