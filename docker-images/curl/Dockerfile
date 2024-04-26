ARG TAG=main
ARG DOCKER_REPOSITORY
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

RUN apk update && \
    apk add openssl curl

COPY scyllaridae.yml /app/scyllaridae.yml
