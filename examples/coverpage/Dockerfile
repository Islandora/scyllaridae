ARG TAG=main
ARG DOCKER_REPOSITORY=lehighlts
FROM ${DOCKER_REPOSITORY}/scyllaridae-pandoc:${TAG}

RUN apt-get update \
      && apt-get install -y --no-install-recommends \
      ghostscript=10.02.1~dfsg1-0ubuntu7.4 \
      && rm -rf /var/lib/apt/lists/*
COPY . /app
