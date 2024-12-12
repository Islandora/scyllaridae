ARG TAG=main
ARG DOCKER_REPOSITORY=lehighlts
FROM islandora/imagemagick:alpine-3.20.2-imagemagick-7.1.1.36-r0 AS imagemagick
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

SHELL ["/bin/ash", "-o", "pipefail", "-c"]

# hadolint ignore=DL3018
RUN --mount=type=bind,from=imagemagick,source=/packages,target=/packages \
    --mount=type=bind,from=imagemagick,source=/etc/apk/keys,target=/etc/apk/keys \
    apk add --no-cache /packages/imagemagick-*.apk jq==1.7.1-r0

RUN magick -list format | grep "JPEG-2000"

ENV OPENAI_MODEL=gpt-4o \
    CHAT_PROMPT="changeme" \
    MAX_TOKENS=16384

COPY . /app/
