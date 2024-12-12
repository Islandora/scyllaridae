#!/usr/bin/env bash

set -eou pipefail

TMP_DIR=$(mktemp -d)
HOCR_URL="$1"
DOMAIN=$(echo "$HOCR_URL"| awk -F/ '{print $1"//"$3}')
# our hOCR filenames are the node ID
NID=$(echo "$HOCR_URL" | xargs basename | awk -F '.hocr' '{print $1}')

# take the base prompt and move it into place
cp /app/prompt.txt "$TMP_DIR/prompt.txt"

# the hOCR document is being streamed into this script
# append the hOCR document into the prompt
# since we're asking the LLM to improve the hOCR doc
cat >> "$TMP_DIR/prompt.txt"
CHAT_PROMPT=$(jq --null-input --rawfile rawstring "$TMP_DIR/prompt.txt" '$rawstring')

# find the service file
SERVICE_FILE_PATH=$(curl -s "$DOMAIN/node/$NID/service-file" | jq -r '.[0].file')
# convert service file to jpg
curl -s "${DOMAIN}${SERVICE_FILE_PATH}" | magick - "$TMP_DIR/img.jpg"

# chatgpt needs it base64 encoded
BASE64_IMAGE=$(base64 -w 0 "$TMP_DIR/img.jpg")

cat <<EOF > "$TMP_DIR/payload.json"
{
  "model": "$OPENAI_MODEL",
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": $CHAT_PROMPT
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "data:image/jpeg;base64,$BASE64_IMAGE"
          }
        }
      ]
    }
  ],
  "max_tokens": $MAX_TOKENS
}
EOF

curl -s https://api.openai.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d "@$TMP_DIR/payload.json" | jq -r .choices[0].message.content

rm -rf "$TMP_DIR"
