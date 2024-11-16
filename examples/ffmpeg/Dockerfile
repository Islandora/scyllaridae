ARG TAG=main
ARG DOCKER_REPOSITORY=lehighlts
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

RUN apk update && \
    apk add --no-cache ffmpeg==6.1.1-r8

COPY scyllaridae.yml /app/scyllaridae.yml
