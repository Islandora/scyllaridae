ARG TAG=main
ARG DOCKER_REPOSITORY=lehighlts
FROM ${DOCKER_REPOSITORY}/scyllaridae-pandoc:${TAG}

# renovate: datasource=repology depName=ubuntu_24_04/ghostscript
ENV GHOSTSCRIPT_VERSION="10.02.1~dfsg1-0ubuntu7.4"

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      ghostscript="${GHOSTSCRIPT_VERSION}" && \
    rm -rf /var/lib/apt/lists/*
COPY . /app
