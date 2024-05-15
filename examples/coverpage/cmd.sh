#!/usr/bin/env bash

set -eou pipefail

convert_unicode_to_latex() {
    local input="$1"
    local output="$input"

    while IFS= read -r line; do
        unicode_char=$(echo "$line" | cut -d ' ' -f 1)
        latex_command=$(echo "$line" | cut -d ' ' -f 2-)
        output="${output//${unicode_char}/${latex_command}}"
    done < unicode_to_latex.map

    echo "$output" | sed -E "s/\^(\{[^}]*\})/\$\^\1\$/g" | sed -E "s/_\{([^}]*)\}/\$_{\1}\$/g"
}

if [ "$#" -ne 4 ]; then
  echo "Usage: $0 NODE-JSON-URL ORIGINAL-PDF-URL FILE-UPLOAD-URI DESTINATION-URI"
  exit 1
fi

NODE_JSON_URL="$1"
ORIGINAL_PDF_URL="$2"
FILE_UPLOAD_URI="$3"
DESTINATION_URI="$4"
TMP_DIR=$(mktemp -d)

curl -L -s -o "$TMP_DIR/node.json" "${NODE_JSON_URL}?_format=json"
NODE_JSON=$(cat "$TMP_DIR/node.json")

# Decode HTML entities and convert them to text using html2text
TITLE=$(echo "$NODE_JSON" | jq -r '.title[0].value' | html2text -nobs -utf8)
convert_unicode_to_latex "$TITLE" > "$TMP_DIR/title.tex"

# Decode HTML entities and convert them to text using html2text
CITATION=$(echo "$NODE_JSON" | jq -r .citation[0].value | html2text -nobs -utf8)
# Make any URL in the citation an href and convert <i> tags to \textit
CITATION=$(echo "$CITATION" | \
  sed -E 's|(https?://[a-zA-Z0-9./?=_-]+)([.,;!?])|\\\\href{\1}{\1}\2|g' | \
  sed -E 's|<i>([^<]+)</i>|\\\\textit{\1}|g')
convert_unicode_to_latex "$CITATION" > "$TMP_DIR/citation.tex"

TMP_FILE="$TMP_DIR/coverpage.tex"
PDF_FILE="$TMP_DIR/coverpage.pdf"
MERGED_PDF="$TMP_DIR/merged.pdf"
EXISTING_PDF="$TMP_DIR/existing.pdf"

# Create the LaTeX file
sed -e "s|TMP_DIR|${TMP_DIR}|" coverpage.tex > "$TMP_FILE"

# Generate the cover page PDF
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
