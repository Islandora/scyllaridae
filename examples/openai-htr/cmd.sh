#!/usr/bin/env bash

set -eou pipefail

curl https://api.openai.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "Transcribe this image that contains handwritten text. Include all text you see in the image. In your response, say absolutely nothing except the text from the image"
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
    "max_tokens": 50
  }' | jq -r .choices[0].message.content
