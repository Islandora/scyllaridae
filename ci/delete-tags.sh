#!/usr/bin/env bash

set -eou pipefail

if [ "$TAG_PATTERN" = "" ] || [ "$TAG_PATTERN" = "^.*" ] || [ "$TAG_PATTERN" = "^main.*" ]; then
  echo "Not mass deleting"
  exit 1
fi

echo "Deleting tags matching pattern '$TAG_PATTERN' in repository ${DOCKER_IMAGE}"

curl -s -o response.json -u "${DOCKER_USERNAME}:${DOCKER_PASSWORD}" "https://hub.docker.com/v2/repositories/${DOCKER_REPOSITORY}/${DOCKER_IMAGE}/tags?page_size=100"
TAGS=$(jq -r '.results[].name' response.json)
if [ -z "$TAGS" ] || [ "$TAGS" = "null" ]; then
  echo "No tags found or failed to retrieve tags."
  exit 1
fi

curl -s -o token.json -XPOST \
  -H "Content-Type: application/json" \
  -d '{"username": "'"${DOCKER_USERNAME}"'", "password": "'"${DOCKER_PASSWORD}"'"}' \
  "https://hub.docker.com/v2/users/login"

TOKEN=$(jq -r .token token.json)
if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "Unable to auth to dockerhub."
  exit 1
fi

for TAG in $TAGS; do
  if [[ "$TAG" =~ $TAG_PATTERN ]]; then
    echo "Deleting tag $TAG"
    RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE \
               -H "Authorization: bearer ${TOKEN}" \
               "https://hub.docker.com/v2/repositories/${DOCKER_REPOSITORY}/${DOCKER_IMAGE}/tags/$TAG/")
    if [ "$RESPONSE" -eq 204 ]; then
      echo "Tag $TAG deleted successfully."
    else
      echo "Failed to delete tag $TAG. Response code: $RESPONSE"
    fi
  fi
done

rm response.json token.json
