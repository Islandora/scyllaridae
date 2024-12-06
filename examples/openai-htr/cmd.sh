#!/usr/bin/env bash

set -eou pipefail

TMP_DIR=$(mktemp -d)

# convert service file to jpg
magick - "$TMP_DIR/img.jpg"

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
          "text": "$PROMPT"
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
