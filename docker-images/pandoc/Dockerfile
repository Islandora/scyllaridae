ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG} AS scyllaridae
FROM pandoc/core:3.1.1 AS pandoc

WORKDIR /app
COPY --from=scyllaridae /app/scyllaridae /app/scyllaridae
COPY scyllaridae.yml .
ENTRYPOINT ["/app/scyllaridae"]
