ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

RUN apk update && \
    apk add --no-cache curl==8.5.0-r0

COPY scyllaridae.yml /app/scyllaridae.yml
