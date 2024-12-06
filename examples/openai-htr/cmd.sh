#!/usr/bin/env bash

set -eou pipefail

curl https://api.openai.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "'"$OPENAI_MODEL"'",
    "messages": [
      {
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "'"$PROMPT"'"
          },
          {
            "type": "image_url",
            "image_url": {
              "url": "'"$1"'"
            }
          }
        ]
      }
    ],
    "max_tokens": '"$MAX_TOKENS"'
  }' | jq -r .choices[0].message.content
