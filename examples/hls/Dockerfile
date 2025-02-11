ARG TAG=main
ARG DOCKER_REPOSITORY=lehighlts
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

# renovate: datasource=repology depName=alpine_3_20/ffmpeg
ENV FFMPEG_VERSION="6.1.1-r8"
# renovate: datasource=repology depName=alpine_3_20/jq
ENV JQ_VERSION="1.7.1-r0"

RUN apk update && \
    apk add --no-cache \
      ffmpeg=="${FFMPEG_VERSION}" \
      jq=="${JQ_VERSION}"

COPY scyllaridae.yml /app/scyllaridae.yml
COPY cmd.sh /app/cmd.sh
