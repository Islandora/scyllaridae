ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

RUN apk update && \
    apk add --no-cache imagemagick==7.1.1.32-r2

COPY scyllaridae.yml /app/scyllaridae.yml
