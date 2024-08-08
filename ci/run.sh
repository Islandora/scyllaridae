#!/usr/bin/env bash

set -eou pipefail

SCRIPT_DIR=$(dirname "$(realpath "$0")")
cd "$SCRIPT_DIR"
docker compose up -d 2>&1 > /dev/null

docker exec ci-test-1 apk update 2>&1 > /dev/null
docker exec ci-test-1 apk add bash curl file 2>&1 > /dev/null
docker exec ci-test-1 /test.sh
echo $?

docker compose down 2>&1 > /dev/null
docker compose rm 2>&1 > /dev/null