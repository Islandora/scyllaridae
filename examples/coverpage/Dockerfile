ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG} AS scyllaridae

RUN apk update && \
    apk add --no-cache \
      jq==1.7.1-r0 \
      texlive==20230506.66984-r3 \
      ghostscript==10.02.1-r0

COPY . .

ENTRYPOINT ["/app/scyllaridae"]
