ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM ${DOCKER_REPOSITORY}/scyllaridae-pandoc:${TAG}

RUN apk add --no-cache \
      bash==5.2.21-r0 \
      curl==8.5.0-r0 \
      ghostscript==10.04.0-r0 \
      jq==1.7.1-r0

COPY . /app
