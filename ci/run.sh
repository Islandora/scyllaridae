#!/usr/bin/env bash

set -eou pipefail

SCRIPT_DIR=$(dirname "$(realpath "$0")")
cd "$SCRIPT_DIR"
docker compose up -d

docker exec ci-test-1 apk update
docker exec ci-test-1 apk add bash curl file
docker exec ci-test-1 /test.sh
echo $?

docker compose down
docker compose rm