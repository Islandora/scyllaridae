ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG} AS scyllaridae
FROM pandoc/latex:3.2.0 AS pandoc

WORKDIR /app
COPY --from=scyllaridae /app/scyllaridae /app/scyllaridae
COPY scyllaridae.yml .
ENTRYPOINT ["/app/scyllaridae"]
