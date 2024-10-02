#!/usr/bin/env bash

set -eou pipefail

export DOCKER_DEFAULT_PLATFORM=linux/amd64

docker build --build-arg=TAG=main --build-arg=DOCKER_REPOSITORY=$DOCKER_REPOSITORY -t $DOCKER_REPOSITORY/scyllaridae:main .

for EXAMPLE in "$@"; do
  docker build --build-arg=TAG=main --build-arg=DOCKER_REPOSITORY=$DOCKER_REPOSITORY -t $DOCKER_REPOSITORY/scyllaridae-$EXAMPLE:main examples/$EXAMPLE
done
