ARG TAG=main
ARG DOCKER_REPOSITORY=lehighlts
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

RUN apk update && \
    apk add --no-cache jq==1.7.1-r0 grep==3.11-r0

COPY cmd.sh /app/
COPY scyllaridae.yml /app/scyllaridae.yml
