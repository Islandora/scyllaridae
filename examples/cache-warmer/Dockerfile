ARG TAG=main
ARG DOCKER_REPOSITORY=lehighlts
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

# renovate: datasource=repology depName=alpine_3_20/jq
ENV JQ_VERSION="1.7.1-r0"
# renovate: datasource=repology depName=alpine_3_20/grep
ENV GREP_VERSION="3.11-r0"

RUN apk update && \
    apk add --no-cache \
      jq=="${JQ_VERSION}" \
      grep=="${GREP_VERSION}"

COPY cmd.sh /app/
COPY scyllaridae.yml /app/scyllaridae.yml
