ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

RUN apk update && \
    apk add --no-cache jq==1.7.1-r0

COPY cmd.sh /app/
COPY scyllaridae.yml /app/scyllaridae.yml
