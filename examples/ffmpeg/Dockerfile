ARG TAG=main
ARG DOCKER_REPOSITORY=lehighlts
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

# renovate: datasource=repology depName=alpine_3_20/ffmpeg
ENV FFMPEG_VERSION="6.1.1-r8"

RUN apk update && \
    apk add --no-cache ffmpeg=="${FFMPEG_VERSION}"

COPY scyllaridae.yml /app/scyllaridae.yml
COPY cmd.sh /app/cmd.sh
