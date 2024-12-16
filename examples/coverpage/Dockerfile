ARG TAG=main
ARG DOCKER_REPOSITORY=lehighlts
FROM ${DOCKER_REPOSITORY}/scyllaridae-pandoc:${TAG}

RUN apt-get update \
      && apt-get install -y --no-install-recommends \
      curl=8.5.0-2ubuntu10.6 \
      ghostscript=10.02.1~dfsg1-0ubuntu7.4 \
      jq=1.7.1-3build1 \
      && rm -rf /var/lib/apt/lists/*
COPY . /app
